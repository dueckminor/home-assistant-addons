package gateway

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/dueckminor/home-assistant-addons/go/auth"
	"github.com/dueckminor/home-assistant-addons/go/services/dns"
	"github.com/dueckminor/home-assistant-addons/go/services/homeassistant"
	"github.com/dueckminor/home-assistant-addons/go/services/smtp"
	"github.com/gin-gonic/gin"
)

type Endpoints struct {
	Gateway *Gateway
}

// CheckHomeAssistantAuth validates Home Assistant authentication headers
// Allows GET requests for any authenticated HA user
// Requires admin privileges for all other HTTP methods (POST, PUT, DELETE, etc.)
// Headers checked: X-Remote-User-Id, X-Remote-User-Name, X-Remote-User-Display-Name, X-Hass-Source
func (ep *Endpoints) CheckHomeAssistantAuth(c *gin.Context) {
	userID := c.GetHeader("X-Remote-User-Id")
	username := c.GetHeader("X-Remote-User-Name")
	displayName := c.GetHeader("X-Remote-User-Display-Name")
	hassSource := c.GetHeader("X-Hass-Source")

	// Check if user is authenticated via Home Assistant ingress
	if userID == "" || hassSource != "core.ingress" {
		c.AbortWithStatusJSON(401, gin.H{"error": "Home Assistant authentication required"})
		return
	}

	// Store user info in context for use by endpoints
	c.Set("ha_user_id", userID)
	c.Set("ha_username", username)
	c.Set("ha_display_name", displayName)

	// For now, treat all authenticated users as admins since we don't have an admin flag
	// You may want to implement admin detection based on userID or username
	c.Set("ha_is_admin", true)

	// For non-GET methods, require admin privileges (currently all users are treated as admin)
	// TODO: Implement proper admin detection if needed
	if c.Request.Method != "GET" {
		// Currently allowing all authenticated users for non-GET operations
		// You can add admin checking logic here if needed
	}

	c.Next()
}

// RequireAuthServer ensures the auth server is available before accessing user/group endpoints
func (ep *Endpoints) RequireAuthServer(c *gin.Context) {
	if ep.Gateway.authServer == nil {
		c.AbortWithStatusJSON(503, gin.H{"error": "Authentication server not available"})
		return
	}
	c.Next()
}

func (ep *Endpoints) setupEndpoints(r *gin.RouterGroup) {
	// DNS endpoints
	r.GET("/dns/external/ipv4", ep.GET_ExternalIpv4)
	r.POST("/dns/external/ipv4", ep.POST_ExternalIpv4)
	r.GET("/dns/external/ipv6", ep.GET_ExternalIpv6)
	r.POST("/dns/external/ipv6", ep.POST_ExternalIpv6)
	r.GET("/dns/ipv4", ep.GET_Ipv4)
	r.GET("/dns/ipv6", ep.GET_Ipv6)
	r.GET("/dns/lookup", ep.GET_DnsLookup)

	// Domain management endpoints
	r.GET("/domains", ep.GET_Domains)
	r.POST("/domains", ep.POST_Domains)
	r.DELETE("/domains/:guid", ep.DELETE_Domains)

	// Route management endpoints
	r.GET("/domains/:guid/routes", ep.GET_DomainsGuidRoutes)
	r.POST("/domains/:guid/routes", ep.POST_DomainsGuidRoutes)
	r.DELETE("/domains/:guid/routes/:rguid", ep.DELETE_DomainsGuidRoutesGuid)
	r.PUT("/domains/:guid/routes/:rguid", ep.PUT_DomainsGuidRoutesGuid)

	// User management endpoints (require both HA auth and auth server availability)
	r.GET("/users", ep.RequireAuthServer, ep.GET_Users)
	r.POST("/users", ep.RequireAuthServer, ep.POST_Users)
	r.DELETE("/users/:guid", ep.RequireAuthServer, ep.DELETE_UsersGuid)
	r.POST("/users/:guid/password_reset", ep.RequireAuthServer, ep.POST_UsersGuidPasswordReset)

	// Group management endpoints (require both HA auth and auth server availability)
	r.GET("/groups", ep.RequireAuthServer, ep.GET_Groups)
	r.POST("/groups", ep.RequireAuthServer, ep.POST_Groups)
	r.DELETE("/groups/:guid", ep.RequireAuthServer, ep.DELETE_GroupsGuid)

	// Mail configuration endpoints
	r.GET("/mail/config", ep.GET_MailConfig)
	r.PUT("/mail/config", ep.PUT_MailConfig)
	r.POST("/mail/test", ep.POST_MailTest)

	// Add-on discovery endpoints
	r.GET("/addons/running", ep.GET_AddonsDiscovery)
	r.GET("/addons/:slug", ep.GET_AddonInfo)

	// InfluxDB integration status
	r.GET("/influxdb/status", ep.GET_InfluxDBStatus)
	r.GET("/influxdb/config", ep.GET_InfluxDBConfig)
	r.PUT("/influxdb/config", ep.PUT_InfluxDBConfig)

	// Debug endpoint to inspect Home Assistant headers
	r.GET("/debug/headers", ep.GET_DebugHeaders)
}

type CertificateInfo struct {
	ValidNotBefore time.Time `json:"valid_not_before"`
	ValidNotAfter  time.Time `json:"valid_not_after"`
}
type DomainWithStatus struct {
	ConfigDomain      `json:",inline"`
	ServerCertificate *CertificateInfo `json:"server_certificate"`
}

func (ep *Endpoints) GET_Domains(c *gin.Context) {
	domainsWithStatus := make([]DomainWithStatus, 0, len(ep.Gateway.config.Domains))
	for _, domain := range ep.Gateway.config.Domains {
		domainWithStatus := DomainWithStatus{
			ConfigDomain: *domain,
		}
		if domain.serverCertificate != nil {
			chain := domain.serverCertificate.GetChain()
			if chain != nil {
				domainWithStatus.ServerCertificate = &CertificateInfo{
					ValidNotBefore: chain[0].OBJ().NotBefore,
					ValidNotAfter:  chain[0].OBJ().NotAfter,
				}
			}
		}
		domainsWithStatus = append(domainsWithStatus, domainWithStatus)
	}

	c.JSON(200, gin.H{"domains": domainsWithStatus})
}

func (ep *Endpoints) POST_Domains(c *gin.Context) {
	var domain ConfigDomain
	c.BindJSON(&domain)

	domain, err := ep.Gateway.AddDomain(domain)

	if err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	c.JSON(200, domain)
}

func (ep *Endpoints) DELETE_Domains(c *gin.Context) {
	guid := c.Param("guid")

	err := ep.Gateway.DelDomain(guid)

	if err != nil {
		c.JSON(404, gin.H{"error": err.Error()})
		return
	}

	c.JSON(200, gin.H{"domains": ep.Gateway.config.Domains})
}

func (ep *Endpoints) GET_ExternalIpv4(c *gin.Context) {
	ext := ep.Gateway.ExternalIPv4()
	if ext == nil {
		c.AbortWithStatusJSON(http.StatusNotFound, gin.H{"error": "external IPv4 not configured"})
		return
	}
	ip, err := ext.Refresh()
	if ip == nil {
		c.AbortWithStatusJSON(http.StatusUnprocessableEntity, gin.H{"error": err.Error()})
		return
	}
	c.JSON(200, gin.H{"method": ep.Gateway.config.Dns.ExternalIpv4.Method, "param": ep.Gateway.config.Dns.ExternalIpv4.Param, "address": ip.String(), "timestamp": time.Now()})

}

func (ep *Endpoints) POST_ExternalIpv4(c *gin.Context) {
	body := struct {
		Test   bool
		Method string
		Param  string
	}{}
	c.BindJSON(&body)

	var externalIp dns.ExternalIP

	testOnly := body.Test
	method := body.Method
	param := body.Param

	switch method {
	case "dns":
		externalIp = dns.NewExternalIP("ip4", param)
	}

	ip, err := externalIp.Refresh()
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if !testOnly {
		ep.Gateway.config.Dns.ExternalIpv4.Method = method
		ep.Gateway.config.Dns.ExternalIpv4.Param = param
		ep.Gateway.SetExternalIPv4(externalIp)
		ep.Gateway.config.save()

	}

	c.JSON(200, gin.H{"method": method, "param": param, "address": ip.String(), "timestamp": time.Now()})
}
func (ep *Endpoints) GET_ExternalIpv6(c *gin.Context) {
	ext := ep.Gateway.ExternalIPv6()
	if ext == nil {
		c.AbortWithStatusJSON(http.StatusNotFound, gin.H{"error": "external IPv6 not configured"})
		return
	}
	ip, err := ext.Refresh()
	if ip == nil {
		c.AbortWithStatusJSON(http.StatusUnprocessableEntity, gin.H{"error": err.Error()})
		return
	}
	c.JSON(200, gin.H{"method": ep.Gateway.config.Dns.ExternalIpv6.Method, "param": ep.Gateway.config.Dns.ExternalIpv6.Param, "address": ip.String(), "timestamp": time.Now()})
}
func (ep *Endpoints) POST_ExternalIpv6(c *gin.Context) {
	body := struct {
		Test   bool
		Method string
		Param  string
	}{}
	c.BindJSON(&body)

	var externalIp dns.ExternalIP

	externalIp, err := ep.Gateway.CreateExternalIPv6(body.Method, body.Param)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ip, err := externalIp.Refresh()
	if ip == nil {
		c.AbortWithStatusJSON(http.StatusUnprocessableEntity, gin.H{"error": err.Error()})
		return
	}

	if !body.Test {
		ep.Gateway.config.Dns.ExternalIpv6.Method = body.Method
		ep.Gateway.config.Dns.ExternalIpv6.Param = body.Param
		ep.Gateway.SetExternalIPv6(externalIp)
		ep.Gateway.config.save()
	}

	c.JSON(200, gin.H{"method": body.Method, "param": body.Param, "address": ip.String(), "timestamp": time.Now()})
}

func (ep *Endpoints) GET_Ipv4(c *gin.Context) {
	hostname := c.Query("hostname")
	if hostname == "" {
		c.JSON(400, gin.H{"error": "hostname parameter is required"})
		return
	}

	ips, err := net.LookupIP(hostname)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}

	for _, ip := range ips {
		if ip.To4() != nil { // IPv4
			c.JSON(200, gin.H{"ip": ip.String(), "timestamp": time.Now()})
			return
		}
	}
	c.JSON(404, gin.H{"error": "No IPv4 found"})
}

func (ep *Endpoints) GET_Ipv6(c *gin.Context) {
	hostname := c.Query("hostname")
	if hostname == "" {
		c.JSON(400, gin.H{"error": "hostname parameter is required"})
		return
	}

	ips, err := net.LookupIP(hostname)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}

	for _, ip := range ips {
		if ip.To4() == nil { // IPv6
			c.JSON(200, gin.H{"ip": ip.String(), "timestamp": time.Now()})
			return
		}
	}
	c.JSON(404, gin.H{"error": "No IPv6 found"})
}

func (ep *Endpoints) lookup(ctx context.Context, address string, ipv6 bool) (resolved string, err error) {
	ips, err := net.LookupIP(address)
	if err != nil {
		return "", err
	}

	for _, ip := range ips {
		if (ip.To4() == nil) == ipv6 { // IPv6
			return ip.String(), nil
		}
	}
	return "", fmt.Errorf("failed to resolve %s", address)
}

// GET_DnsLookup performs DNS lookups for various record types
func (ep *Endpoints) GET_DnsLookup(c *gin.Context) {
	hostname := c.Query("hostname")
	recordType := c.Query("type")

	// Create DNS client with 5 second timeout
	dnsClient := dns.NewDNSClient(5 * time.Second)

	// Perform the DNS lookup
	result := dnsClient.LookupDNS(hostname, recordType)

	// Handle errors
	if result.Error != "" {
		if hostname == "" {
			c.JSON(400, gin.H{"error": result.Error})
		} else if strings.Contains(result.Error, "Unsupported record type") {
			c.JSON(400, gin.H{"error": result.Error})
		} else if strings.Contains(result.Error, "No records found") || strings.Contains(result.Error, "No DNS response received") {
			c.JSON(404, gin.H{"error": result.Error, "type": result.Type})
		} else {
			c.JSON(500, gin.H{"error": result.Error})
		}
		return
	}

	// Return successful result
	c.JSON(200, result)
}

func (ep *Endpoints) GET_DomainsGuidRoutes(c *gin.Context) {
	guid := c.Param("guid")
	for _, domain := range ep.Gateway.config.Domains {
		if domain.Guid == guid {
			c.JSON(200, gin.H{"routes": domain.Routes})
			return
		}
	}
}

func (ep *Endpoints) POST_DomainsGuidRoutes(c *gin.Context) {
	guid := c.Param("guid")

	var route ConfigRoute
	c.BindJSON(&route)

	route, err := ep.Gateway.AddRoute(guid, route)
	if err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}
	c.JSON(200, route)
}

func (ep *Endpoints) DELETE_DomainsGuidRoutesGuid(c *gin.Context) {
	guid := c.Param("guid")
	rguid := c.Param("rguid")

	err := ep.Gateway.DelRoute(guid, rguid)
	if err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}
	c.JSON(200, gin.H{"status": "deleted"})
}

func (ep *Endpoints) PUT_DomainsGuidRoutesGuid(c *gin.Context) {
	guid := c.Param("guid")
	rguid := c.Param("rguid")

	var route ConfigRoute
	c.BindJSON(&route)

	route, err := ep.Gateway.UpdateRoute(guid, rguid, route)
	if err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}
	c.JSON(200, route)
}

func (ep *Endpoints) GET_Users(c *gin.Context) {
	users := ep.Gateway.authServer.Users()
	c.JSON(200, gin.H{"users": users.Users()})
}

func (ep *Endpoints) POST_Users(c *gin.Context) {
	var user auth.User
	c.BindJSON(&user)

	var err error
	user, err = ep.Gateway.authServer.Users().AddUser(user)
	if err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	c.JSON(200, user)
}

func (ep *Endpoints) DELETE_UsersGuid(c *gin.Context) {
	guid := c.Param("guid")

	err := ep.Gateway.authServer.Users().DeleteUser(guid)
	if err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}
	c.JSON(200, gin.H{"status": "deleted"})
}

func (ep *Endpoints) POST_UsersGuidPasswordReset(c *gin.Context) {
	guid := c.Param("guid")

	user, err := ep.Gateway.authServer.Users().GetUser(guid)
	if err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	client := ep.Gateway.GetSMTPClient()
	if client == nil {
		c.JSON(400, gin.H{"error": "Mail service is not enabled"})
		return
	}
	client.SendWelcomeEmail(user.Mail, user.Name)
}

func (ep *Endpoints) GET_Groups(c *gin.Context) {
	users := ep.Gateway.authServer.Users()
	c.JSON(200, gin.H{"groups": users.Groups()})
}

func (ep *Endpoints) POST_Groups(c *gin.Context) {
	var group auth.Group
	c.BindJSON(&group)

	var err error
	group, err = ep.Gateway.authServer.Users().AddGroup(group.Name)
	if err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	c.JSON(200, group)
}

func (ep *Endpoints) DELETE_GroupsGuid(c *gin.Context) {
	guid := c.Param("guid")

	err := ep.Gateway.authServer.Users().DeleteGroup(guid)
	if err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}
	c.JSON(200, gin.H{"status": "deleted"})
}

func (ep *Endpoints) GET_MailConfig(c *gin.Context) {
	config := ep.Gateway.config.Mail
	// For security reasons, mask the password if it's set
	if config.Password != "" {
		config.Password = "-"
	}
	c.JSON(200, config)
}

func (ep *Endpoints) PUT_MailConfig(c *gin.Context) {
	var mailConfig ConfigMail
	if err := c.ShouldBindJSON(&mailConfig); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	// Handle password security logic
	if mailConfig.Password == "-" {
		// If Email or SmtpHost have been changed, reset the password
		if ep.Gateway.config.Mail.Email != mailConfig.Email || ep.Gateway.config.Mail.SmtpHost != mailConfig.SmtpHost {
			mailConfig.Password = ""
		} else {
			// Keep the existing password unchanged
			mailConfig.Password = ep.Gateway.config.Mail.Password
		}
	}

	ep.Gateway.config.Mail = mailConfig
	if err := ep.Gateway.config.save(); err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}

	// Return the config with masked password for security
	responseConfig := ep.Gateway.config.Mail
	if responseConfig.Password != "" {
		responseConfig.Password = "-"
	}

	c.JSON(200, responseConfig)
}

func (ep *Endpoints) POST_MailTest(c *gin.Context) {
	var testRequest struct {
		Email string `json:"email"`
	}
	if err := c.ShouldBindJSON(&testRequest); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	if !ep.Gateway.config.Mail.Enabled {
		c.JSON(400, gin.H{"error": "Mail service is not enabled"})
		return
	}

	smtpClient := smtp.NewClient(smtp.Config{
		Host:     ep.Gateway.config.Mail.SmtpHost,
		Port:     ep.Gateway.config.Mail.SmtpPort,
		Username: ep.Gateway.config.Mail.Email,
		Password: ep.Gateway.config.Mail.Password,
		UseTLS:   ep.Gateway.config.Mail.UseTLS,
	})

	// Send test email
	message := &smtp.Message{
		From:    ep.Gateway.config.Mail.FromEmail,
		To:      []string{testRequest.Email},
		Subject: "Gateway Mail Configuration Test",
		Body:    "This is a test email from your Gateway mail configuration. If you receive this, your mail settings are working correctly!",
		BodyHTML: `
			<h2>Gateway Mail Configuration Test</h2>
			<p>This is a test email from your Gateway mail configuration.</p>
			<p><strong>If you receive this, your mail settings are working correctly!</strong></p>
			<hr>
			<p><small>Sent by Gateway Mail Service</small></p>
		`,
	}

	if err := smtpClient.SendMail(message); err != nil {
		c.JSON(500, gin.H{"error": "Failed to send test email: " + err.Error()})
		return
	}

	c.JSON(200, gin.H{"status": "Test email sent successfully"})
}

// GET_DebugHeaders returns all request headers for debugging Home Assistant integration
func (ep *Endpoints) GET_DebugHeaders(c *gin.Context) {
	headers := make(map[string]string)
	for name, values := range c.Request.Header {
		if len(values) > 0 {
			headers[name] = values[0]
		}
	}

	// Also include context values set by our auth middleware
	haUserID, exists := c.Get("ha_user_id")
	if exists {
		headers["_context_ha_user_id"] = haUserID.(string)
	}

	haUsername, exists := c.Get("ha_username")
	if exists {
		headers["_context_ha_username"] = haUsername.(string)
	}

	haIsAdmin, exists := c.Get("ha_is_admin")
	if exists {
		headers["_context_ha_is_admin"] = fmt.Sprintf("%t", haIsAdmin.(bool))
	}

	c.JSON(200, gin.H{
		"headers": headers,
		"method":  c.Request.Method,
		"path":    c.Request.URL.Path,
	})
}

// GET_AddonsDiscovery returns all running add-ons with their network details
func (e *Endpoints) GET_AddonsDiscovery(c *gin.Context) {
	supervisorClient := homeassistant.NewSupervisorClient()

	targets, err := supervisorClient.GetRunningAddons()
	if err != nil {
		c.JSON(500, gin.H{"error": "Failed to discover add-ons: " + err.Error()})
		return
	}

	c.JSON(200, gin.H{
		"result": "ok",
		"data":   targets,
	})
}

// GET_AddonInfo returns detailed information for a specific add-on
func (e *Endpoints) GET_AddonInfo(c *gin.Context) {
	slug := c.Param("slug")
	if slug == "" {
		c.JSON(400, gin.H{"error": "Add-on slug is required"})
		return
	}

	supervisorClient := homeassistant.NewSupervisorClient()

	info, err := supervisorClient.GetAddonInfo(slug)
	if err != nil {
		c.JSON(500, gin.H{"error": "Failed to get add-on info: " + err.Error()})
		return
	}

	c.JSON(200, gin.H{
		"result": "ok",
		"data":   info,
	})
}

// GET_InfluxDBStatus returns the current InfluxDB integration status
func (e *Endpoints) GET_InfluxDBStatus(c *gin.Context) {
	if e.Gateway.influxDBConfig == nil || !e.Gateway.influxDBConfig.Found {
		c.JSON(200, gin.H{
			"result": "ok",
			"data": gin.H{
				"enabled": false,
				"message": "No InfluxDB add-on detected",
			},
		})
		return
	}

	// Return sanitized configuration (no password)
	c.JSON(200, gin.H{
		"result": "ok",
		"data": gin.H{
			"enabled":  true,
			"name":     e.Gateway.influxDBConfig.Name,
			"slug":     e.Gateway.influxDBConfig.Slug,
			"url":      e.Gateway.influxDBConfig.URL,
			"database": e.Gateway.influxDBConfig.Database,
			"username": e.Gateway.influxDBConfig.Username,
			"message":  "InfluxDB integration active",
		},
	})
}

// GET_InfluxDBConfig returns the InfluxDB configuration (credentials)
func (e *Endpoints) GET_InfluxDBConfig(c *gin.Context) {
	config := e.Gateway.config.InfluxDB
	// Mask password for security
	maskedPassword := ""
	if config.Password != "" {
		maskedPassword = "********"
	}

	c.JSON(200, gin.H{
		"username": config.Username,
		"password": maskedPassword,
	})
}

// PUT_InfluxDBConfig updates the InfluxDB credentials
func (e *Endpoints) PUT_InfluxDBConfig(c *gin.Context) {
	var newConfig ConfigInfluxDB
	if err := c.BindJSON(&newConfig); err != nil {
		c.JSON(400, gin.H{"error": "Invalid request body"})
		return
	}

	// Update the configuration
	e.Gateway.config.InfluxDB = newConfig

	// Also update the active InfluxDB config if detected
	if e.Gateway.influxDBConfig != nil && e.Gateway.influxDBConfig.Found {
		e.Gateway.influxDBConfig.Username = newConfig.Username
		e.Gateway.influxDBConfig.Password = newConfig.Password
	}

	// Save configuration
	e.Gateway.config.save()

	// Return masked password
	maskedPassword := ""
	if newConfig.Password != "" {
		maskedPassword = "********"
	}

	c.JSON(200, gin.H{
		"username": newConfig.Username,
		"password": maskedPassword,
		"message":  "InfluxDB credentials updated successfully",
	})
}
