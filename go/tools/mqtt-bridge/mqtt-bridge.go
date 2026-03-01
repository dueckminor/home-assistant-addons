package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"path"
	"sync"
	"syscall"

	"github.com/dueckminor/home-assistant-addons/go/addons/mqttbridge"
	"github.com/dueckminor/home-assistant-addons/go/services/automation"
	"github.com/dueckminor/home-assistant-addons/go/services/homeassistant"
	"github.com/dueckminor/home-assistant-addons/go/services/homematic"
	"github.com/dueckminor/home-assistant-addons/go/services/influxdb"
	"github.com/dueckminor/home-assistant-addons/go/services/mqtt"
	"github.com/dueckminor/home-assistant-addons/go/utils/crypto/rand"
	"gopkg.in/yaml.v3"
)

var dataDir string
var adminPort int
var distAdmin string

var theConfig BrigeConfig

func init() {
	flag.StringVar(&dataDir, "data-dir", "/data", "the data dir")
	flag.IntVar(&adminPort, "admin-port", 8080, "the port for the admin-ui")
	flag.StringVar(&distAdmin, "dist-admin", "", "the URL for the admin-ui")
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

type InfluxDbConfig struct {
	InfluxDbUri      string `yaml:"influx_db_uri"`
	InfluxDbUser     string `yaml:"influx_db_user"`
	InfluxDbPassword string `yaml:"influx_db_password"`
}

type BrigeConfig struct {
	MqttConfig                `yaml:",inline"`
	InfluxDbConfig            `yaml:",inline"`
	homematic.HomematicConfig `yaml:",inline"`
}
type influxMeasurement struct {
	config    homeassistant.Config
	tags      map[string]string
	available *influxAvailability
	state     string
}

type influxAvailability struct {
	measurements map[string]*influxMeasurement
	available    bool
}

func toInflux(influxConfig InfluxDbConfig) {
	influx, err := influxdb.NewClient(influxConfig.InfluxDbUri, "ha", influxConfig.InfluxDbUser, influxConfig.InfluxDbPassword)
	if err != nil {
		panic(err)
	}
	mqttbridge.EnableInflux(influx)
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

	mqttClientId := "mqtt-bridge-" + id

	fmt.Println("MQTT URI:", theConfig.MqttURI)
	fmt.Println("MQTT Client ID:", mqttClientId)

	mqttBroker := mqtt.NewBroker(theConfig.MqttURI, theConfig.MqttUser, theConfig.MqttPassword)
	mqttConn, err := mqttBroker.Dial(mqttClientId, "")
	if err != nil {
		panic(err)
	}

	go mqttbridge.Listen(ctx, mqttConn)

	automation.GetRegistry().EnableMqtt(mqttBroker)

	// Create server first so we can pass it to other functions
	s := mqttbridge.NewServer(adminPort, distAdmin)
	s.SetMqttConn(mqttConn)

	if theConfig.InfluxDbUri != "" {
		toInflux(theConfig.InfluxDbConfig)
	}

	if theConfig.HomematicUri != "" {
		homematic.StartMqttBridge(theConfig.HomematicConfig)
	}

	err = s.Listen()
	if err != nil {
		panic(err)
	}

	wg.Go(func() {
		s.Serve(ctx)
	})

	wg.Go(func() {
		<-ctx.Done()
	})

	wg.Wait()

	fmt.Println("DONE!!!")
}
