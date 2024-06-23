package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"path"
	"strings"
	"sync"
	"syscall"

	"github.com/dueckminor/home-assistant-addons/go/acme"
	"github.com/dueckminor/home-assistant-addons/go/dns"
	"github.com/dueckminor/home-assistant-addons/go/homeassistant"
	"github.com/dueckminor/home-assistant-addons/go/httpbin"
	"github.com/dueckminor/home-assistant-addons/go/network"
	"github.com/dueckminor/home-assistant-addons/go/pki"
	"gopkg.in/yaml.v3"
)

var dataDir string
var dnsPort int
var httpPort int
var httpsPort int

type configExternalIp struct {
	Source string `yaml:"source"`
	Entity string `yaml:"entity"`
}

type configDev struct {
	Domain string `yaml:"domain"`
	Ip     string `yaml:"ip"`
}

type configServer struct {
	Hostname string `yaml:"hostname"`
	Target   string `yaml:"target"`
	Mode     string `yaml:"mode"`
}

type config struct {
	Domains    []string         `yaml:"domains"`
	ExternalIp configExternalIp `yaml:"external_ip"`
	Servers    []configServer   `yaml:"servers"`
	Dev        configDev        `yaml:"dev"`
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

func configureServer(proxy network.TLSProxy, server configServer) {
	if strings.HasPrefix(server.Target, "http://") || strings.HasPrefix(server.Target, "https://") {
		options := network.ParseReverseProxyOptions(server.Mode)
		proxy.AddHandler(server.Hostname, network.NewHostImplReverseProxy(server.Target, options))
	}
	if strings.HasPrefix(server.Target, "insecure-https://") {
		options := network.ParseReverseProxyOptions(server.Mode)
		proxy.AddHandler(server.Hostname, network.NewHostImplReverseProxy(server.Target[9:], options, network.ReverseProxyOptions{InsecureTLS: true}))
	}
	if strings.HasPrefix(server.Target, "tcp://") {
		proxy.AddHandler(server.Hostname, network.NewDialTCPRaw("tcp", server.Target[6:]))
	}
}

func configureServers(proxy network.TLSProxy, servers []configServer) {
	for _, server := range servers {
		configureServer(proxy, server)
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

	var extIp string
	var err error
	if theConfig.ExternalIp.Source == "httpbin" {
		extIp, err = httpbin.NewAPI().GetExternalIp()
	} else {
		haApi := homeassistant.NewAPI()
		extIp, err = haApi.GetEntityValue("sensor.fritz_box_7590_externe_ip")
	}
	if err != nil {
		panic(err)
	}
	fmt.Println("External IP:", extIp)

	dnsServer, err := dns.NewServer(fmt.Sprintf(":%d", dnsPort))
	if err != nil {
		panic(err)
	}
	dnsServer.SetExternalIp(extIp)
	dnsServer.AddDomains(theConfig.Domains...)

	//dnsServer.AddDevDomain(extIp, theConfig.Dev.Ip, theConfig.Dev.Domain)

	acmeClient, err := acme.NewClient(dataDir, dnsServer)
	if err != nil {
		panic(err)
	}

	httpsServer, err := network.NewTLSProxy()
	if err != nil {
		panic(err)
	}

	serverCertificates := make([]pki.ServerCertificate, 0)

	for _, domain := range theConfig.Domains {
		serverCertificate := pki.NewServerCertificate(path.Join(dataDir, domain), acmeClient, "*."+domain)
		serverCertificate.SetTLSServer(httpsServer)
		serverCertificates = append(serverCertificates, serverCertificate)
	}

	configureServers(httpsServer, theConfig.Servers)

	wg.Add(1)
	go func() {
		err := httpsServer.ListenAndServe(ctx, "tcp", fmt.Sprintf(":%d", httpsPort))
		if err != nil {
			fmt.Println(err)
		}
		wg.Done()
	}()

	wg.Wait()

	fmt.Println("gateway stopped...")
}
