package main

import (
	"context"
	"flag"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path"
	"sync"
	"syscall"

	addon "github.com/dueckminor/home-assistant-addons/go/addons/alphaess"
	"github.com/dueckminor/home-assistant-addons/go/embed/alphaess_dist"
	"github.com/dueckminor/home-assistant-addons/go/utils/ginutil"
	"github.com/gin-gonic/gin"
	"gopkg.in/yaml.v3"
)

var dataDir string
var adminPort int
var distDir string
var configDir string

var theConfig addon.AlphaEssConfig

func init() {
	flag.StringVar(&dataDir, "data-dir", "/data", "the data dir")
	flag.StringVar(&configDir, "config-dir", "/homeassistant", "the config dir")
	flag.IntVar(&adminPort, "admin-port", 8080, "the port for the admin-ui")
	flag.StringVar(&distDir, "dist", "", "the URL for the admin-ui")
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

	alphaEssAddon := addon.NewAddon(addon.AlphaEssAddonConfig{
		AlphaEssConfig:         theConfig,
		DataDir:                dataDir,
		HomeAssistantConfigDir: configDir,
	})

	// Setup web server
	r := gin.Default()

	if distDir != "" {
		ginutil.ServeFromUri(r, distDir)
	} else {
		ginutil.ServeEmbedFS(r, alphaess_dist.FS, "dist")
	}

	api := r.Group("/api")

	endpoints := alphaEssAddon.Endpoints()
	endpoints.SetupEndpoints(api)

	httpServer := &http.Server{
		Addr:    fmt.Sprintf(":%d", adminPort),
		Handler: r,
	}

	listener, err := net.Listen("tcp", httpServer.Addr)
	if err != nil {
		panic(err)
	}

	fmt.Println("Web server listening on:", httpServer.Addr)

	// Start web server
	wg.Add(1)
	go func() {
		defer wg.Done()
		go func() {
			<-ctx.Done()
			httpServer.Shutdown(context.Background())
		}()
		httpServer.Serve(listener)
	}()

	wg.Add(1)
	go func() {
		<-ctx.Done()
		wg.Done()
	}()

	wg.Wait()

	fmt.Println("AlphaESS addon stopped.")
}
