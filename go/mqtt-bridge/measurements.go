package mqttbridge

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/dueckminor/home-assistant-addons/go/services/homeassistant"
	"github.com/dueckminor/home-assistant-addons/go/services/influxdb"
)

type Measurement struct {
	Config            homeassistant.Config
	AvailabilityTopic TopicRef
	StateTopic        TopicRef
	ConfigTopic       TopicRef
	Tags              map[string]string
}

var influx influxdb.Client

func EnableInflux(influxClient influxdb.Client) {
	influx = influxClient
}

func (m *Measurement) onChange() {
	state := m.StateTopic.Get().Value
	fmt.Println(m.Tags["device"], m.Tags["entity_id"], state, m.Config.UnitOfMeasurement)

	if influx != nil {
		value, _ := strconv.ParseFloat(state, 64)
		influx.SendMetric(m.Config.UnitOfMeasurement, value, m.Tags)
	}
}

var (
	measurements = make(map[string]*Measurement)
)

func createOrUpdateMeasurement(topicRef TopicRef, getOrCreateTopic func(name string) TopicRef) *Measurement {
	topic := topicRef.Get()
	var config homeassistant.Config
	config.Unmarshal([]byte(topic.Value))

	measurement, ok := measurements[config.UniqueId]
	if !ok {
		measurement = &Measurement{
			Config: config,
			Tags:   make(map[string]string),
		}
		measurements[config.UniqueId] = measurement

		topicParts := strings.Split(topic.Name, "/")

		measurement.Tags["device_class"] = config.DeviceClass
		measurement.Tags["domain"] = "sensor"
		measurement.Tags["device"] = topicParts[len(topicParts)-3]
		measurement.Tags["entity_id"] = config.Name

	}
	if measurement.ConfigTopic == nil {
		measurement.ConfigTopic = topicRef
	}
	if measurement.StateTopic == nil && config.StateTopic != "" {
		stateTopic := getOrCreateTopic(config.StateTopic)
		stateTopic.AddHandler(func(t Topic) {
			measurement.onChange()
		})
		measurement.StateTopic = stateTopic
	}
	if measurement.AvailabilityTopic == nil && config.AvailabilityTopic != "" {
		measurement.AvailabilityTopic = getOrCreateTopic(config.AvailabilityTopic)
	}
	return measurement
}
