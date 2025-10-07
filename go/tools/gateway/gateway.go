package main

import (
	"context"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"path"
	"strings"
	"sync"
	"syscall"

	"github.com/dueckminor/home-assistant-addons/go/acme"
	"github.com/dueckminor/home-assistant-addons/go/auth"
	"github.com/dueckminor/home-assistant-addons/go/dns"
	"github.com/dueckminor/home-assistant-addons/go/gatewayconfig"
	"github.com/dueckminor/home-assistant-addons/go/network"
	"github.com/dueckminor/home-assistant-addons/go/pki"
	"github.com/gin-gonic/gin"
	"gopkg.in/yaml.v3"
)

var dataDir string
var dnsPort int
var httpPort int
var httpsPort int

type configExternalIp struct {
	Source  string `yaml:"source"`
	Options string `yaml:"options"`
}

type configServer struct {
	Hostname string `yaml:"hostname"`
	Target   string `yaml:"target"`
	Mode     string `yaml:"mode"`
}

type configDev struct {
	Domain      string `yaml:"domain"`
	HttpTarget  string `yaml:"http_target"`
	HttpsTarget string `yaml:"https_target"`
	DnsTarget   string `yaml:"dns_target"`
}

type config struct {
	Domains      []string         `yaml:"domains"`
	ExternalIp   configExternalIp `yaml:"external_ip"`
	ExternalIpv6 configExternalIp `yaml:"external_ipv6"`
	Servers      []configServer   `yaml:"servers"`
	Dev          configDev        `yaml:"dev"`
}

var theConfig config

func init() {
	flag.StringVar(&dataDir, "data-dir", "/data", "the data dir")
	flag.IntVar(&dnsPort, "dns-port", 53, "the DNS port")
	flag.IntVar(&httpPort, "http-port", 80, "the HTTP port")
	flag.IntVar(&httpsPort, "https-port", 443, "the HTTPS port")
	flag.Parse()

	configFile := path.Join(dataDir, "options.json")
	configJson, err := os.ReadFile(configFile)
	if err != nil {
		if !os.IsNotExist(err) {
			panic(err)
		}
	}

	err = yaml.Unmarshal(configJson, &theConfig)
	if err != nil {
		panic(err)
	}
}

type Gateway struct {
	authServer *auth.AuthServer
	authClient *auth.AuthClient
}

func (g *Gateway) configureServer(proxy network.TLSProxy, server configServer) {
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

func (g *Gateway) configureAuthServer(proxy network.TLSProxy, server configServer) {
	if g.authServer != nil {
		panic("only one auth-server allowed")
	}
	r := gin.Default()
	var err error
	g.authServer, err = auth.NewAuthServer(r, os.Getenv("DIST_AUTH"), path.Join(dataDir, "auth"))
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

func (g *Gateway) configureServers(proxy network.TLSProxy, servers []configServer) {
	for _, server := range servers {
		if server.Target == "@auth" {
			g.configureAuthServer(proxy, server)
		}
	}
	for _, server := range servers {
		if server.Target != "@auth" {
			g.configureServer(proxy, server)
		}
	}
}

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	wg := sync.WaitGroup{}

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		sig := <-sigs
		fmt.Println()
		fmt.Println(sig)
		cancel()
	}()

	fmt.Println("gateway started...")
	fmt.Println("External-IP-Source:", theConfig.ExternalIp.Source)
	var err error

	var extIPv4 dns.ExternalIP
	var extIPv6 dns.ExternalIP

	switch theConfig.ExternalIp.Source {
	case "dns":
		extIPv4 = dns.NewExternalIP("ip4", theConfig.ExternalIp.Options)
	}
	switch theConfig.ExternalIpv6.Source {
	case "dns":
		extIPv6 = dns.NewExternalIP("ip6", theConfig.ExternalIpv6.Options)
	}

	dnsServer, err := dns.NewServer(fmt.Sprintf(":%d", dnsPort))
	if err != nil {
		panic(err)
	}
	dnsServer.SetExternalIP(extIPv4)
	dnsServer.SetExternalIPv6(extIPv6)
	dnsServer.AddDomains(theConfig.Domains...)

	acmeClient, err := acme.NewClient(dataDir, dnsServer)
	if err != nil {
		panic(err)
	}

	g := new(Gateway)

	httpServer := network.NewHttpToHttps()

	httpsServer, err := network.NewTLSProxy()
	if err != nil {
		panic(err)
	}

	for _, domain := range theConfig.Domains {
		serverCertificate := pki.NewServerCertificate(path.Join(dataDir, domain), acmeClient, "*."+domain)
		serverCertificate.SetTLSServer(httpsServer)
	}

	if theConfig.Dev.Domain != "" {
		if theConfig.Dev.HttpsTarget != "" {
			httpsServer.InternalOnly("*." + theConfig.Dev.Domain)
			g.configureServer(httpsServer, configServer{
				Hostname: "*." + theConfig.Dev.Domain,
				Target:   theConfig.Dev.HttpsTarget,
				Mode:     "raw",
			})
		}
		if theConfig.Dev.DnsTarget != "" {
			dnsServer.AddProxyDomain(theConfig.Dev.Domain, theConfig.Dev.DnsTarget)
		}
	}

	g.configureServers(httpsServer, theConfig.Servers)

	wg.Add(1)
	go func() {
		err := httpServer.ListenAndServe(ctx, "tcp", fmt.Sprintf(":%d", httpPort))
		if err != nil {
			fmt.Println(err)
		}
		wg.Done()
	}()

	wg.Add(1)
	go func() {
		err := httpsServer.ListenAndServe(ctx, "tcp", fmt.Sprintf(":%d", httpsPort))
		if err != nil {
			fmt.Println(err)
		}
		wg.Done()
	}()

	wg.Add(1)
	go func() {
		r := gin.Default()

		gatewayconfig.NewGatewayConfigServer(r, os.Getenv("DIST_CONFIG"))

		fmt.Println("Starting configuration server on port 8099...")
		configServer := &http.Server{
			Addr:    ":8099",
			Handler: r,
		}
		err := configServer.ListenAndServe()
		if err != nil {
			fmt.Printf("Config server error: %v\n", err)
		}
		wg.Done()
	}()

	wg.Wait()

	fmt.Println("gateway stopped...")
}
