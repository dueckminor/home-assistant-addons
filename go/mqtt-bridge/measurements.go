package mqttbridge

import (
	"fmt"
	"strings"

	"github.com/dueckminor/home-assistant-addons/go/services/homeassistant"
)

type Measurement struct {
	Config            homeassistant.Config
	AvailabilityTopic TopicRef
	StateTopic        TopicRef
	ConfigTopic       TopicRef
	Tags              map[string]string
}

func (m *Measurement) onChange() {
	fmt.Println(m.StateTopic.Get().Name, m.StateTopic.Get().Value)
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
