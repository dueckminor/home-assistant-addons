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

	"github.com/dueckminor/home-assistant-addons/go/crypto/rand"
	"github.com/dueckminor/home-assistant-addons/go/services/homeassistant"
	"github.com/dueckminor/home-assistant-addons/go/services/mqtt"
	"gopkg.in/yaml.v3"
)

var dataDir string

var theConfig BrigeConfig

func init() {
	flag.StringVar(&dataDir, "data-dir", "/data", "the data dir")
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

type BrigeLegacyConfig struct {
	MqttConfig     `yaml:",inline"`
	InfluxDbConfig `yaml:",inline"`
}

type BrigeConfig struct {
	MqttConfig     `yaml:",inline"`
	InfluxDbConfig `yaml:",inline"`
	Legacy         *BrigeLegacyConfig `yaml:"legacy"`
}

func fromLegacyMqtt(mqttConn mqtt.Conn, legacyConfig MqttConfig, mqttClientId string) {
	mqttLegacyBroker := mqtt.NewBroker(legacyConfig.MqttURI, legacyConfig.MqttUser, legacyConfig.MqttPassword)

	mqttLegacyConn, err := mqttLegacyBroker.Dial(mqttClientId, "")
	if err != nil {
		panic(err)
	}
	mqttLegacyConn.Forward("#", mqttConn)
}

func toInflux(mqttConn mqtt.Conn, influxConfig InfluxDbConfig) {
	type influxMeasurement struct {
		config    homeassistant.Config
		available bool
		state     string
	}

	uniqueIds := make(map[string]*influxMeasurement)
	availabilityTopics := make(map[string]*influxMeasurement)
	stateTopics := make(map[string]*influxMeasurement)

	mqttConn.Subscribe("homeassistant/sensor/#", func(topic, payload string) {
		if strings.HasSuffix(topic, "/config") {
			var config homeassistant.Config
			config.Unmarshal([]byte(payload))
			measurement := uniqueIds[config.UniqueId]
			if measurement == nil {
				measurement = &influxMeasurement{
					config:    config,
					available: false,
				}
				uniqueIds[config.UniqueId] = measurement
				availabilityTopics[config.AvailabilityTopic] = measurement
				stateTopics[config.StateTopic] = measurement
			}
		}
	})

	mqttConn.Subscribe("#", func(topic, payload string) {
		if measurement, ok := availabilityTopics[topic]; ok {
			fmt.Println("availability_topic:", topic)
			measurement.available = payload == "online"
			return
		}
		if measurement, ok := stateTopics[topic]; ok {
			fmt.Println("state_topic:", topic, payload, measurement.config.UnitOfMeasurement)
			measurement.state = payload
			return
		}
	})
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

	mqttBroker := mqtt.NewBroker(theConfig.MqttURI, theConfig.MqttUser, theConfig.MqttPassword)
	mqttConn, err := mqttBroker.Dial(mqttClientId, "")
	if err != nil {
		panic(err)
	}

	if theConfig.Legacy != nil && theConfig.Legacy.MqttURI != "" {
		fromLegacyMqtt(mqttConn, theConfig.Legacy.MqttConfig, mqttClientId)
	}

	if theConfig.InfluxDbUri != "" {
		toInflux(mqttConn, theConfig.InfluxDbConfig)
	}

	wg.Add(1)
	go func() {
		<-ctx.Done()
		wg.Done()
	}()

	wg.Wait()
}
