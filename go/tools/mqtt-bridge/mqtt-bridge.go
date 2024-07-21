package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"path"
	"strconv"
	"strings"
	"sync"
	"syscall"

	"github.com/dueckminor/home-assistant-addons/go/crypto/rand"
	"github.com/dueckminor/home-assistant-addons/go/services/alphaess"
	"github.com/dueckminor/home-assistant-addons/go/services/automation"
	"github.com/dueckminor/home-assistant-addons/go/services/homeassistant"
	"github.com/dueckminor/home-assistant-addons/go/services/influxdb"
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
	AlphaEssUri    string             `yaml:"alphaess_uri"`
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

func toInflux(mqttConn mqtt.Conn, influxConfig InfluxDbConfig) {

	influx, err := influxdb.NewClient(influxConfig.InfluxDbUri, "ha", influxConfig.InfluxDbUser, influxConfig.InfluxDbPassword)
	if err != nil {
		panic(err)
	}

	uniqueIds := make(map[string]*influxMeasurement)
	availabilityTopics := make(map[string]*influxAvailability)
	stateTopics := make(map[string]*influxMeasurement)

	mqttConn.Subscribe("homeassistant/sensor/#", func(topic, payload string) {
		if strings.HasSuffix(topic, "/config") {
			var config homeassistant.Config
			config.Unmarshal([]byte(payload))
			measurement := uniqueIds[config.UniqueId]
			if measurement == nil {
				measurement = &influxMeasurement{
					config: config,
					tags:   make(map[string]string),
				}

				topicParts := strings.Split(topic, "/")

				measurement.tags["device_class"] = config.DeviceClass
				measurement.tags["domain"] = "sensor"
				measurement.tags["device"] = topicParts[len(topicParts)-3]
				measurement.tags["entity_id"] = config.Name

				a := availabilityTopics[config.AvailabilityTopic]
				if a == nil {
					a = &influxAvailability{
						measurements: make(map[string]*influxMeasurement),
					}
					availabilityTopics[config.AvailabilityTopic] = a
				}
				measurement.available = a

				uniqueIds[config.UniqueId] = measurement
				stateTopics[config.StateTopic] = measurement
				a.measurements[config.UniqueId] = measurement
			}
		}
	})

	mqttConn.Subscribe("#", func(topic, payload string) {
		if a, ok := availabilityTopics[topic]; ok {
			fmt.Println("availability_topic:", topic)
			available := payload == "online"
			if a.available == available {
				return
			}
			a.available = available
			if !available {
				return
			}
			for _, m := range a.measurements {
				if m.state != "" {
					fmt.Println(m.tags["device"], m.tags["entity_id"], m.state, m.config.UnitOfMeasurement)
					value, _ := strconv.ParseFloat(m.state, 64)
					influx.SendMetric(m.config.UnitOfMeasurement, value, m.tags)
				}
			}
			return
		}
		if m, ok := stateTopics[topic]; ok {
			m.state = payload
			if m.available.available {
				fmt.Println(m.tags["device"], m.tags["entity_id"], m.state, m.config.UnitOfMeasurement)
				value, _ := strconv.ParseFloat(m.state, 64)
				influx.SendMetric(m.config.UnitOfMeasurement, value, m.tags)
			}
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

	if theConfig.AlphaEssUri != "" {
		automation.GetRegistry().EnableMqtt(mqttBroker)
		automation.GetRegistry().EnableHomeAssistant()
		alphaess.Run(theConfig.AlphaEssUri)
	}

	wg.Add(1)
	go func() {
		<-ctx.Done()
		wg.Done()
	}()

	wg.Wait()
}
