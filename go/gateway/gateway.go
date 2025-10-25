package gateway

import (
	"context"
	"embed"
	"fmt"
	"net/http"
	"path"
	"strings"
	"sync"

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

	debug bool
}

func (g *Gateway) EnableDebugMode() {
	g.debug = true
}

func (g *Gateway) Wait() {
	g.wg.Wait()
}

func (g *Gateway) Start(ctx context.Context, dnsPort int, httpPort int, httpsPort int, configPort int) (err error) {
	ctx, g.cancel = context.WithCancel(ctx)

	defer func() {
		if err != nil {
			g.cancel()
		}
	}()

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
	}

	return nil
}

func (g *Gateway) startRoute(route *ConfigRoute) {
	hostname := route.GetHostname()

	if strings.HasPrefix(route.Target, "http://") || strings.HasPrefix(route.Target, "https://") {
		options := network.ReverseProxyOptions{
			UseTargetHostname: route.Options.UseTargetHostname,
			InsecureTLS:       route.Options.Insecure,
			Auth:              route.Options.Auth,
			AuthSecret:        route.Options.AuthSecret,
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
	g.httpServer = network.NewHttpToHttps()

	g.wg.Add(1)

	go func() {
		defer func() {
			g.httpServer = nil
			g.wg.Done()
		}()
		err := g.httpServer.ListenAndServe(ctx, "tcp", fmt.Sprintf(":%d", port))
		if err != nil {
			fmt.Println(err)
		}
	}()

	return nil
}

func (g *Gateway) StartHttpsServer(ctx context.Context, port int) (err error) {
	g.httpsServer, err = network.NewTLSProxy()
	if err != nil {
		return err
	}

	g.wg.Add(1)

	go func() {
		defer func() {
			g.httpsServer = nil
			g.wg.Done()
		}()
		err := g.httpsServer.ListenAndServe(ctx, "tcp", fmt.Sprintf(":%d", port))
		if err != nil {
			fmt.Println(err)
		}
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
