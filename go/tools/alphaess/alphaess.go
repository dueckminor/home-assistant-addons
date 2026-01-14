package main

import (
	"context"
	"embed"
	"flag"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path"
	"sync"
	"syscall"

	"github.com/dueckminor/home-assistant-addons/go/crypto/rand"
	"github.com/dueckminor/home-assistant-addons/go/ginutil"
	"github.com/dueckminor/home-assistant-addons/go/services/alphaess"
	"github.com/dueckminor/home-assistant-addons/go/services/automation"
	"github.com/dueckminor/home-assistant-addons/go/services/mqtt"
	"github.com/gin-gonic/gin"
	"gopkg.in/yaml.v3"
)

//go:embed dist/*
var distFS embed.FS

var dataDir string
var adminPort int
var distDir string

var theConfig AlphaEssConfig

func init() {
	flag.StringVar(&dataDir, "data-dir", "/data", "the data dir")
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

type MqttConfig struct {
	MqttURI      string `yaml:"mqtt_uri"`
	MqttUser     string `yaml:"mqtt_user"`
	MqttPassword string `yaml:"mqtt_password"`
}

type AlphaEssConfig struct {
	MqttConfig  `yaml:",inline"`
	AlphaEssUri string `yaml:"alphaess_uri"`
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

	id, err := rand.GetString(10)
	if err != nil {
		panic(err)
	}

	mqttClientId := "alphaess-" + id

	fmt.Println("MQTT URI:", theConfig.MqttURI)
	fmt.Println("MQTT Client ID:", mqttClientId)
	fmt.Println("AlphaESS URI:", theConfig.AlphaEssUri)

	if theConfig.AlphaEssUri == "" {
		fmt.Println("AlphaESS URI not configured, exiting...")
		return
	}

	mqttBroker := mqtt.NewBroker(theConfig.MqttURI, theConfig.MqttUser, theConfig.MqttPassword)
	mqttConn, err := mqttBroker.Dial(mqttClientId, "")
	if err != nil {
		panic(err)
	}
	defer mqttConn.Close()

	automation.GetRegistry().EnableMqtt(mqttBroker)
	automation.GetRegistry().EnableHomeAssistant()

	// Setup web server
	r := gin.Default()

	if distDir != "" {
		ginutil.ServeFromUri(r, distDir)
	} else {
		ginutil.ServeEmbedFS(r, distFS, "dist")
	}

	// API endpoints
	api := r.Group("/api")
	api.GET("/status", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"connected":         true,
			"mqttConnected":     true,
			"alphaessConnected": true,
			"mqttUri":           theConfig.MqttURI,
			"alphaessUri":       theConfig.AlphaEssUri,
		})
	})

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

	// Start AlphaESS integration
	alphaess.Run(theConfig.AlphaEssUri)

	wg.Add(1)
	go func() {
		<-ctx.Done()
		wg.Done()
	}()

	wg.Wait()

	fmt.Println("AlphaESS addon stopped.")
}
