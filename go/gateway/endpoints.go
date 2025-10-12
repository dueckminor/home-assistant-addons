package gateway

import (
	"context"
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/dueckminor/home-assistant-addons/go/auth"
	"github.com/dueckminor/home-assistant-addons/go/dns"
	"github.com/dueckminor/home-assistant-addons/go/smtp"
	"github.com/gin-gonic/gin"
)

type Endpoints struct {
	Gateway *Gateway
}

func (ep *Endpoints) setupEndpoints(r *gin.RouterGroup) {
	r.GET("/dns/external/ipv4", ep.GET_ExternalIpv4)
	r.POST("/dns/external/ipv4", ep.POST_ExternalIpv4)
	r.GET("/dns/external/ipv6", ep.GET_ExternalIpv6)
	r.POST("/dns/external/ipv6", ep.POST_ExternalIpv6)
	r.GET("/dns/ipv4", ep.GET_Ipv4)
	r.GET("/dns/ipv6", ep.GET_Ipv6)
	r.GET("/dns/lookup", ep.GET_DnsLookup)

	r.GET("/domains", ep.GET_Domains)
	r.POST("/domains", ep.POST_Domains)
	r.DELETE("/domains/:guid", ep.DELETE_Domains)

	r.GET("/domains/:guid/routes", ep.GET_DomainsGuidRoutes)
	r.POST("/domains/:guid/routes", ep.POST_DomainsGuidRoutes)
	r.DELETE("/domains/:guid/routes/:rguid", ep.DELETE_DomainsGuidRoutesGuid)
	r.PUT("/domains/:guid/routes/:rguid", ep.PUT_DomainsGuidRoutesGuid)

	r.GET("/users", ep.Check_Users, ep.GET_Users)
	r.POST("/users", ep.Check_Users, ep.POST_Users)
	r.DELETE("/users/:guid", ep.Check_Users, ep.DELETE_UsersGuid)
	r.POST("/users/:guid/password_reset", ep.Check_Users, ep.POST_UsersGuidPasswordReset)

	r.GET("/groups", ep.Check_Users, ep.GET_Groups)
	r.POST("/groups", ep.Check_Users, ep.POST_Groups)
	r.DELETE("/groups/:guid", ep.Check_Users, ep.DELETE_GroupsGuid)

	r.GET("/mail/config", ep.GET_MailConfig)
	r.PUT("/mail/config", ep.PUT_MailConfig)
	r.POST("/mail/test", ep.POST_MailTest)
}

func (ep *Endpoints) GET_Domains(c *gin.Context) {
	c.JSON(200, gin.H{"domains": ep.Gateway.config.Domains})
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
	addr, err := ep.lookup(c.Request.Context(), ep.Gateway.config.Dns.ExternalIpv4.Options, false)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	c.JSON(200, gin.H{"method": ep.Gateway.config.Dns.ExternalIpv4.Source, "source": ep.Gateway.config.Dns.ExternalIpv4.Options, "address": addr, "timestamp": time.Now()})
}

func (ep *Endpoints) POST_ExternalIpv4(c *gin.Context) {
	var h gin.H
	c.BindJSON(&h)
	ep.Gateway.config.Dns.ExternalIpv4.Source = "dns"
	ep.Gateway.config.Dns.ExternalIpv4.Options = h["source"].(string)

	extIPv4 := dns.NewExternalIP("ip4", ep.Gateway.config.Dns.ExternalIpv4.Options)
	ip := extIPv4.ExternalIP()
	ep.Gateway.dnsServer.SetExternalIP(extIPv4)
	ep.Gateway.config.save()

	c.JSON(200, gin.H{"method": ep.Gateway.config.Dns.ExternalIpv4.Source, "source": ep.Gateway.config.Dns.ExternalIpv4.Options, "address": ip.String(), "timestamp": time.Now()})
}
func (ep *Endpoints) GET_ExternalIpv6(c *gin.Context) {
	addr, err := ep.lookup(c.Request.Context(), ep.Gateway.config.Dns.ExternalIpv6.Options, true)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	c.JSON(200, gin.H{"method": ep.Gateway.config.Dns.ExternalIpv6.Source, "source": ep.Gateway.config.Dns.ExternalIpv6.Options, "address": addr, "timestamp": time.Now()})
}
func (ep *Endpoints) POST_ExternalIpv6(c *gin.Context) {
	var h gin.H
	c.BindJSON(&h)
	ep.Gateway.config.Dns.ExternalIpv6.Source = "dns"
	ep.Gateway.config.Dns.ExternalIpv6.Options = h["source"].(string)
	extIPv6 := dns.NewExternalIP("ip6", ep.Gateway.config.Dns.ExternalIpv6.Options)
	ip := extIPv6.ExternalIP()
	ep.Gateway.dnsServer.SetExternalIPv6(extIPv6)
	ep.Gateway.config.save()

	c.JSON(200, gin.H{"method": ep.Gateway.config.Dns.ExternalIpv6.Source, "source": ep.Gateway.config.Dns.ExternalIpv6.Options, "address": ip.String(), "timestamp": time.Now()})
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

func (ep *Endpoints) Check_Users(c *gin.Context) {
	if ep.Gateway.authServer == nil {
		c.AbortWithStatusJSON(503, gin.H{"error": "Authentication server not available"})
	}
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
