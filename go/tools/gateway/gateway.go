package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/dueckminor/home-assistant-addons/go/gateway"
)

var data string
var distGateway string
var distAuth string
var dnsPort int
var httpPort int
var httpsPort int

func init() {
	flag.StringVar(&data, "data", "/data", "the data dir")
	flag.StringVar(&distGateway, "dist-gateway", "", "the dist dir for the gateway (or uri)")
	flag.StringVar(&distAuth, "dist-auth", "", "the dist dir for the auth (or uri)")
	flag.IntVar(&dnsPort, "dns-port", 53, "the DNS port")
	flag.IntVar(&httpPort, "http-port", 80, "the HTTP port")
	flag.IntVar(&httpsPort, "https-port", 443, "the HTTPS port")
	flag.Parse()
}

func main() {
	ctx, cancel := context.WithCancel(context.Background())

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		sig := <-sigs
		fmt.Println()
		fmt.Println(sig)
		cancel()
	}()

	gw, err := gateway.NewGateway(data, distGateway, distAuth)
	if err != nil {
		panic(err)
	}

	err = gw.Start(ctx, dnsPort, httpPort, httpsPort, 8099)
	if err != nil {
		panic(err)
	}

	fmt.Println("gateway started...")

	gw.Wait()

	fmt.Println("gateway stopped...")
}
