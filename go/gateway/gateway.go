package gateway

import (
	"context"
	"embed"
	"fmt"
	"io"
	"net/http"
	"os"
	"path"
	"strings"
	"sync"
	"time"

	"github.com/dueckminor/home-assistant-addons/go/acme"
	"github.com/dueckminor/home-assistant-addons/go/auth"
	"github.com/dueckminor/home-assistant-addons/go/dns"
	"github.com/dueckminor/home-assistant-addons/go/ginutil"
	"github.com/dueckminor/home-assistant-addons/go/network"
	"github.com/dueckminor/home-assistant-addons/go/pki"
	"github.com/dueckminor/home-assistant-addons/go/services/homeassistant"
	"github.com/dueckminor/home-assistant-addons/go/smtp"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

//go:embed dist/*
var distFS embed.FS

func NewGateway(dataDir string, distGateway string, distAuth string) (g *Gateway, err error) {

	configFile := path.Join(dataDir, "config.yml")

	g = &Gateway{
		distGateway: distGateway,
		distAuth:    distAuth,
		dataDir:     dataDir,
	}

	g.config, err = loadConfig(configFile)
	if err != nil {
		return nil, err
	}

	return g, nil
}

type Gateway struct {
	cancel func()
	config *Config

	distGateway string
	distAuth    string
	dataDir     string

	wg sync.WaitGroup

	authServer *auth.AuthServer
	authClient *auth.AuthClient

	dnsServer dns.Server

	acmeClient acme.Client

	httpServer  network.HttpToHttps
	httpsServer network.TLSProxy

	externalIPv4 dns.ExternalIP
	externalIPv6 dns.ExternalIP

	influxDBConfig   *homeassistant.InfluxDBConfig
	metricsCollector *MetricsCollector

	debug bool
}

func (g *Gateway) EnableDebugMode() {
	g.debug = true
}

func (g *Gateway) Wait() {
	g.wg.Wait()
}

func (g *Gateway) detectInfluxDB() {
	// Only attempt detection if running in Home Assistant (SUPERVISOR_TOKEN is set)
	if os.Getenv("SUPERVISOR_TOKEN") == "" {
		fmt.Println("InfluxDB detection skipped: not running in Home Assistant environment")
		return
	}

	supervisorClient := homeassistant.NewSupervisorClient()
	config, err := supervisorClient.DetectInfluxDB()
	if err != nil {
		fmt.Printf("InfluxDB detection error: %v\n", err)
		return
	}

	if config.Found {
		g.influxDBConfig = config

		// Try to get credentials from environment variables (set by add-on options)
		envUsername := os.Getenv("INFLUXDB_USERNAME")
		envPassword := os.Getenv("INFLUXDB_PASSWORD")
		envDatabase := os.Getenv("INFLUXDB_DATABASE")

		if envUsername != "" {
			g.influxDBConfig.Username = envUsername
			g.influxDBConfig.Password = envPassword
		} else if g.config.InfluxDB.Username != "" {
			// Fallback to config file credentials if no env vars
			g.influxDBConfig.Username = g.config.InfluxDB.Username
			g.influxDBConfig.Password = g.config.InfluxDB.Password
		}

		// Override database name if provided
		if envDatabase != "" {
			g.influxDBConfig.Database = envDatabase
		}

		fmt.Printf("✅ InfluxDB detected: %s\n", config.Name)
		fmt.Printf("   URL: %s\n", config.URL)
		fmt.Printf("   Database: %s\n", g.influxDBConfig.Database)
		if g.influxDBConfig.Username != "" {
			fmt.Printf("   Username: %s\n", g.influxDBConfig.Username)
		} else {
			fmt.Println("   ⚠️  No credentials configured - metrics disabled")
			fmt.Println("   Configure in Home Assistant add-on settings")
		}

		// Only send startup metric if credentials are available
		if g.influxDBConfig.Username != "" {
			g.sendStartupMetric()
			g.startMetricsCollector()
		}
	} else {
		fmt.Println("ℹ️  No InfluxDB add-on detected")
	}
}

func (g *Gateway) startMetricsCollector() {
	client, err := g.influxDBConfig.CreateClient()
	if err != nil {
		fmt.Printf("⚠️  Failed to create InfluxDB client for metrics: %v\n", err)
		return
	}

	// Create metrics collector with 1-minute interval
	g.metricsCollector = NewMetricsCollector(client, 1*time.Minute)
	g.metricsCollector.Start()

	fmt.Println("📊 Metrics collector started (reporting every 1 minute)")
}

func (g *Gateway) sendStartupMetric() {
	client, err := g.influxDBConfig.CreateClient()
	if err != nil {
		fmt.Printf("⚠️  Failed to create InfluxDB client: %v\n", err)
		return
	}
	defer client.Close()

	// Send startup event with value 1
	tags := map[string]string{
		"service": "gateway",
		"event":   "startup",
	}

	err = client.SendMetric("gateway_events", 1, tags)
	if err != nil {
		fmt.Printf("⚠️  Failed to send startup metric: %v\n", err)
		return
	}

	fmt.Println("✅ Startup metric sent to InfluxDB successfully")
}

func (g *Gateway) metricCallback(metric network.Metric) {
	if g.metricsCollector != nil {
		g.metricsCollector.RecordMetric(metric)
	}
}

func (g *Gateway) Start(ctx context.Context, dnsPort int, httpPort int, httpsPort int, configPort int) (err error) {
	ctx, g.cancel = context.WithCancel(ctx)

	defer func() {
		if err != nil {
			g.cancel()
		}
	}()

	// Detect InfluxDB add-on at startup
	g.detectInfluxDB()

	err = g.StartDNS(ctx, dnsPort)
	if err == nil {
		err = g.StartHttpServer(ctx, httpPort)
	}
	if err == nil {
		err = g.StartHttpsServer(ctx, httpsPort)
	}
	if err == nil {
		err = g.StartAcmeClient(ctx)
	}
	if err == nil {
		err = g.StartUI(ctx, 8099)
	}

	for _, domain := range g.config.Domains {
		g.startDomain(domain)
	}

	authRoute := g.config.GetAuthRoute()
	if authRoute != nil {
		g.startAuthServer(authRoute)

		smtpClient := g.GetSMTPClient()
		if smtpClient != nil {
			g.authServer.EnableSMTP(authRoute.GetHostname(), authRoute.domain.Name, smtpClient)
		}

	}

	for _, domain := range g.config.Domains {
		for _, route := range domain.Routes {
			if route.Target != "@auth" {
				g.startRoute(route)
			}
		}

		if domain.Redirect != nil && domain.Redirect.Target != "" {
			g.httpServer.SetHandler("supervisor."+domain.Name, http.HandlerFunc(g.redirectToSupervisor))
		}
	}

	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case <-time.After(5 * time.Minute):
				if g.externalIPv4 != nil {
					go g.externalIPv4.Refresh()
				}
				if g.externalIPv6 != nil {
					go g.externalIPv6.Refresh()
				}
			}
		}
	}()

	return nil
}

func (g *Gateway) redirectToSupervisor(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	debugTokenPath := path.Join(g.dataDir, "debug_token")
	debugToken := ""
	if tokenBytes, err := os.ReadFile(debugTokenPath); err == nil {
		debugToken = strings.TrimSpace(string(tokenBytes))
	}
	if debugToken == "" || r.Header.Get("Authorization") != "Bearer "+debugToken {
		http.Error(w, "Supervisor access is not allowed", http.StatusForbidden)
		return
	}

	req, err := http.NewRequestWithContext(r.Context(), r.Method, "http://supervisor"+r.URL.Path, r.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Copy headers from original request
	for name, values := range r.Header {
		for _, value := range values {
			req.Header.Add(name, value)
		}
	}

	req.Header.Set("Authorization", "Bearer "+os.Getenv("SUPERVISOR_TOKEN"))

	// Copy query parameters
	req.URL.RawQuery = r.URL.RawQuery

	// Forward the request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	// Copy response headers
	for name, values := range resp.Header {
		for _, value := range values {
			w.Header().Add(name, value)
		}
	}

	// Copy status code and body
	w.WriteHeader(resp.StatusCode)
	io.Copy(w, resp.Body)
}

func (g *Gateway) startRoute(route *ConfigRoute) {
	hostname := route.GetHostname()

	if strings.HasPrefix(route.Target, "http://") || strings.HasPrefix(route.Target, "https://") {
		options := network.ReverseProxyOptions{
			UseTargetHostname: route.Options.UseTargetHostname,
			InsecureTLS:       route.Options.Insecure,
			Auth:              route.Options.Auth,
			AuthSecret:        route.Options.AuthSecret,
			MetricCallback:    g.metricCallback,
		}
		if options.Auth {
			if g.authClient == nil {
				return
			}
			options.AuthClient = new(auth.AuthClient)
			*options.AuthClient = *g.authClient
			options.AuthClient.Secret = options.AuthSecret
			options.SessionStore = g.authServer.GetSessionStore()
		}
		g.httpsServer.AddHandler(hostname, network.NewHostImplReverseProxy(route.Target, options))
	}
	if strings.HasPrefix(route.Target, "tcp://") {
		g.httpsServer.AddHandler(hostname, network.NewDialTCPRaw("tcp", route.Target[6:]))
	}
}

func (g *Gateway) stopRoute(route *ConfigRoute) {
	hostname := route.GetHostname()
	g.httpsServer.DeleteHandler(hostname)
}

func (g *Gateway) startAuthServer(route *ConfigRoute) {
	if g.authServer != nil {
		panic("only one auth-server allowed")
	}
	r := gin.Default()
	var err error
	g.authServer, err = auth.NewAuthServer(r, g.distAuth, path.Join(g.dataDir, "auth"))
	if err != nil {
		panic(err)
	}

	acc, err := g.authServer.GetAuthClientConfig("gateway")
	if err != nil {
		panic(err)
	}

	hostname := route.GetHostname()

	g.authClient = &auth.AuthClient{
		AuthURI:      "https://" + hostname,
		ClientID:     acc.ClientId,
		ClientSecret: acc.ClientSecret,
		ServerKey:    g.authServer.GetPublicKey(),
		Secret:       "",
	}

	g.httpsServer.AddHandler(hostname, network.NewGinHandler(r))
}

func (g *Gateway) AddDomain(domain ConfigDomain) (ConfigDomain, error) {
	existingDomain := g.config.GetDomainByName(domain.Name)
	if existingDomain != nil {
		return ConfigDomain{}, fmt.Errorf("domain %q already exists", domain.Name)
	}
	domain.Guid = uuid.New().String()

	g.startDomain(&domain)

	for _, route := range domain.Routes {
		route.Guid = uuid.New().String()
		route.domain = &domain
		if route.Target == "@auth" {
			g.startAuthServer(route)
		}
		g.startRoute(route)
	}

	g.config.Domains = append(g.config.Domains, &domain)
	g.config.save()
	return domain, nil
}

func (g *Gateway) DelDomain(guid string) error {
	existingDomain := g.config.DeleteDomain(guid)
	if existingDomain == nil {
		return fmt.Errorf("domain with guid %q not found", guid)
	}

	for _, route := range existingDomain.Routes {
		g.stopRoute(route)
	}
	g.stopDomain(existingDomain)

	g.config.save()
	return nil
}

func (g *Gateway) AddRoute(domainGuid string, route ConfigRoute) (ConfigRoute, error) {
	route.Guid = uuid.New().String()
	domain := g.config.GetDomain(domainGuid)
	if domain == nil {
		return ConfigRoute{}, fmt.Errorf("domain with guid %q not found", domainGuid)
	}
	domain.AddRoute(&route)
	g.startRoute(&route)
	g.config.save()
	return route, nil
}

func (g *Gateway) DelRoute(domainGuid string, routeGuid string) error {
	domain := g.config.GetDomain(domainGuid)
	if domain == nil {
		return fmt.Errorf("domain with guid %q not found", domainGuid)
	}
	route := domain.DeleteRoute(routeGuid)
	if route == nil {
		return fmt.Errorf("route with guid %q not found", routeGuid)
	}
	g.stopRoute(route)
	g.config.save()
	return nil
}

func (g *Gateway) UpdateRoute(domainGuid string, routeGuid string, route ConfigRoute) (ConfigRoute, error) {
	domain := g.config.GetDomain(domainGuid)
	if domain == nil {
		return ConfigRoute{}, fmt.Errorf("domain with guid %q not found", domainGuid)
	}
	existingRoute := domain.GetRoute(routeGuid)
	if existingRoute == nil {
		return ConfigRoute{}, fmt.Errorf("route with guid %q not found", routeGuid)
	}

	existingRoute.Options = route.Options
	if existingRoute.Hostname != route.Hostname {
		g.stopRoute(existingRoute)
		existingRoute.Hostname = route.Hostname
	}
	existingRoute.Target = route.Target
	g.startRoute(existingRoute)
	g.config.save()

	return *existingRoute, nil
}

func (g *Gateway) ExternalIPv4() (extIp dns.ExternalIP) {
	return g.externalIPv4
}

func (g *Gateway) ExternalIPv6() (extIp dns.ExternalIP) {
	return g.externalIPv6
}

func (g *Gateway) SetExternalIPv4(extIp dns.ExternalIP) {
	g.externalIPv4 = extIp
	g.dnsServer.SetExternalIPv4(extIp)
}

func (g *Gateway) SetExternalIPv6(extIp dns.ExternalIP) {
	g.externalIPv6 = extIp
	g.dnsServer.SetExternalIPv6(extIp)
}

func (g *Gateway) CreateExternalIPv4(method, params string) (extIp dns.ExternalIP, err error) {
	switch method {
	case "":
		return nil, nil
	case "dns":
		return dns.NewExternalIP("ip4", params), nil
	}
	return nil, fmt.Errorf("unknown external ipv4 method %q", method)
}

func (g *Gateway) CreateExternalIPv6(method, params string) (extIp dns.ExternalIP, err error) {
	switch method {
	case "":
		return nil, nil
	case "dns":
		return dns.NewExternalIP("ip6", params), nil
	case "homeassistant":
		return homeassistant.NewExternalIP(), nil
	}
	return nil, fmt.Errorf("unknown external ipv6 method %q", method)
}

func (g *Gateway) StartDNS(ctx context.Context, port int) (err error) {
	g.externalIPv4, err = g.CreateExternalIPv4(g.config.Dns.ExternalIpv4.Method, g.config.Dns.ExternalIpv4.Param)
	if err != nil {
		return err
	}
	g.externalIPv6, err = g.CreateExternalIPv6(g.config.Dns.ExternalIpv6.Method, g.config.Dns.ExternalIpv6.Param)
	if err != nil {
		return err
	}

	if g.externalIPv4 != nil {
		_, _ = g.externalIPv4.Refresh()
	}
	if g.externalIPv6 != nil {
		_, _ = g.externalIPv6.Refresh()
	}

	g.dnsServer, err = dns.NewServer(fmt.Sprintf(":%d", port))
	if err != nil {
		panic(err)
	}
	g.dnsServer.SetExternalIPv4(g.externalIPv4)
	g.dnsServer.SetExternalIPv6(g.externalIPv6)

	return nil
}

func (g *Gateway) StartAcmeClient(ctx context.Context) (err error) {
	g.acmeClient, err = acme.NewClient(path.Join(g.dataDir, "acme"), g.dnsServer)
	if err != nil {
		return err
	}
	return nil
}

func (g *Gateway) StartHttpServer(ctx context.Context, port int) (err error) {
	g.httpServer, err = network.NewHttpToHttps("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		return err
	}
	go func() {
		<-ctx.Done()
		g.httpServer.Close()
		g.httpServer = nil
	}()
	return nil
}

func (g *Gateway) StartHttpsServer(ctx context.Context, port int) (err error) {
	g.httpsServer, err = network.NewTLSProxy("tcp", fmt.Sprintf(":%d", port))
	g.httpsServer.SetMetricCallback(g.metricCallback)
	if err != nil {
		return err
	}
	go func() {
		<-ctx.Done()
		g.httpServer.Close()
		g.httpServer = nil
	}()
	return nil
}

func (g *Gateway) startDomain(domain *ConfigDomain) {
	if domain.Redirect != nil && domain.Redirect.Target != "" {
		g.startRedirectDomain(domain)
		return
	}

	g.dnsServer.AddDomains(domain.Name)
	domain.serverCertificate = pki.NewServerCertificate(path.Join(g.acmeClient.DataDir(), domain.Name), g.acmeClient, "*."+domain.Name)
	domain.serverCertificate.SetTLSServer(g.httpsServer)
}

func (g *Gateway) startRedirectDomain(domain *ConfigDomain) {
	g.httpsServer.InternalOnly("*." + domain.Name)
	g.startRoute(&ConfigRoute{
		Hostname: "*",
		Target:   domain.Redirect.GetHTTPSTarget(),
		domain:   domain,
	})
	g.dnsServer.AddProxyDomain(domain.Name, domain.Redirect.GetDNSTarget())
}

func (g *Gateway) stopDomain(domain *ConfigDomain) {
	if domain.Redirect != nil && domain.Redirect.Target != "" {
		g.stopRedirectDomain(domain)
		return
	}

	g.dnsServer.DelDomains(domain.Name)
	domain.serverCertificate.Close()
}

func (g *Gateway) stopRedirectDomain(domain *ConfigDomain) {

}

func (g *Gateway) GetSMTPClient() *smtp.Client {
	if !g.config.Mail.Enabled {
		return nil
	}
	return smtp.NewClient(smtp.Config{
		From:     g.config.Mail.FromEmail,
		Host:     g.config.Mail.SmtpHost,
		Port:     g.config.Mail.SmtpPort,
		Username: g.config.Mail.Email,
		Password: g.config.Mail.Password,
		UseTLS:   g.config.Mail.UseTLS,
	})
}

func (g *Gateway) StartUI(ctx context.Context, port int) error {
	r := gin.Default()

	httpServer := &http.Server{
		Addr:    fmt.Sprintf(":%d", port),
		Handler: r,
	}

	go func() {
		<-ctx.Done()
		httpServer.Shutdown(context.Background())
	}()

	if g.distGateway != "" {
		ginutil.ServeFromUri(r, g.distGateway)
	} else {
		ginutil.ServeEmbedFS(r, distFS, "dist")
	}

	ep := Endpoints{Gateway: g}
	api := r.Group("/api")

	if !g.debug {
		// Apply Home Assistant authentication middleware to all API endpoints
		api.Use(ep.CheckHomeAssistantAuth)
	}

	ep.setupEndpoints(api)

	// Development endpoint to test headers without authentication (for debugging only)
	r.GET("/api/dev/headers", func(c *gin.Context) {
		headers := make(map[string]string)
		for name, values := range c.Request.Header {
			if len(values) > 0 {
				headers[name] = values[0]
			}
		}
		c.JSON(200, gin.H{
			"headers": headers,
			"note":    "This endpoint bypasses authentication - for development only",
		})
	})

	g.wg.Add(1)

	go func() {
		defer func() {
			g.wg.Done()
		}()
		fmt.Printf("Config server started on port %d\n", port)
		err := httpServer.ListenAndServe()
		if err != nil {
			fmt.Printf("Config server error: %v\n", err)
		}
	}()

	return nil
}
