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
	"github.com/gin-gonic/gin"
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

	dnsEndpoints DnsEndpoints

	wg sync.WaitGroup

	authServer *auth.AuthServer
	authClient *auth.AuthClient

	dnsServer dns.Server

	acmeClient acme.Client

	httpServer  network.HttpToHttps
	httpsServer network.TLSProxy
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
		err = g.StartDevSupport(ctx)
	}
	if err == nil {
		err = g.StartUI(ctx, 8099)
	}

	if err == nil {
		g.ConfigureServers(g.config.Servers)
	}

	if len(g.config.Domains) > 0 {
		g.AddDomains(g.config.Domains...)
	}

	return nil
}

func (g *Gateway) ConfigureServer(proxy network.TLSProxy, server ConfigServer) {
	if strings.HasPrefix(server.Target, "http://") || strings.HasPrefix(server.Target, "https://") {
		options := network.ParseReverseProxyOptions(server.Mode)
		if options.Auth {
			options.AuthClient = new(auth.AuthClient)
			*options.AuthClient = *g.authClient
			options.AuthClient.Secret = options.AuthSecret
			options.SessionStore = g.authServer.GetSessionStore()
		}
		proxy.AddHandler(server.Hostname, network.NewHostImplReverseProxy(server.Target, options))
	}
	if strings.HasPrefix(server.Target, "tcp://") {
		proxy.AddHandler(server.Hostname, network.NewDialTCPRaw("tcp", server.Target[6:]))
	}
}

func (g *Gateway) configureAuthServer(proxy network.TLSProxy, server ConfigServer) {
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

	g.authClient = &auth.AuthClient{
		AuthURI:      "https://" + server.Hostname,
		ClientID:     acc.ClientId,
		ClientSecret: acc.ClientSecret,
		ServerKey:    g.authServer.GetPublicKey(),
		Secret:       "",
	}

	proxy.AddHandler(server.Hostname, network.NewGinHandler(r))
}

func (g *Gateway) ConfigureServers(servers []ConfigServer) {
	for _, server := range servers {
		if server.Target == "@auth" {
			g.configureAuthServer(g.httpsServer, server)
		}
	}
	for _, server := range servers {
		if server.Target != "@auth" {
			g.ConfigureServer(g.httpsServer, server)
		}
	}
}

func (g *Gateway) AddDomains(domains ...string) {
	g.dnsServer.AddDomains(domains...)

	for _, domain := range domains {
		serverCertificate := pki.NewServerCertificate(path.Join(g.dataDir, domain), g.acmeClient, "*."+domain)
		serverCertificate.SetTLSServer(g.httpsServer)
	}
}

func (g *Gateway) StartDNS(ctx context.Context, port int) (err error) {
	var extIPv4 dns.ExternalIP
	var extIPv6 dns.ExternalIP

	switch g.config.ExternalIp.Source {
	case "dns":
		extIPv4 = dns.NewExternalIP("ip4", g.config.ExternalIp.Options)
	}
	switch g.config.ExternalIpv6.Source {
	case "dns":
		extIPv6 = dns.NewExternalIP("ip6", g.config.ExternalIpv6.Options)
	}

	g.dnsServer, err = dns.NewServer(fmt.Sprintf(":%d", port))
	if err != nil {
		panic(err)
	}
	g.dnsServer.SetExternalIP(extIPv4)
	g.dnsServer.SetExternalIPv6(extIPv6)

	return nil
}

func (g *Gateway) StartAcmeClient(ctx context.Context) (err error) {
	g.acmeClient, err = acme.NewClient(g.dataDir, g.dnsServer)
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

func (g *Gateway) StartDevSupport(ctx context.Context) (err error) {
	if g.config.Dev.Domain != "" {
		if g.config.Dev.HttpsTarget != "" {
			g.httpsServer.InternalOnly("*." + g.config.Dev.Domain)
			g.ConfigureServer(g.httpsServer, ConfigServer{
				Hostname: "*." + g.config.Dev.Domain,
				Target:   g.config.Dev.HttpsTarget,
				Mode:     "raw",
			})
		}
		if g.config.Dev.DnsTarget != "" {
			g.dnsServer.AddProxyDomain(g.config.Dev.Domain, g.config.Dev.DnsTarget)
		}
	}
	return nil
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

	g.setupEndpoints(r)

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

func (g *Gateway) setupEndpoints(r *gin.Engine) {
	api := r.Group("/api")
	g.dnsEndpoints.server = g.dnsServer
	g.dnsEndpoints.config = g.config
	g.dnsEndpoints.setupEndpoints(api)

	(&Domains{config: g.config}).setupEndpoints(api)
}
