package homeassistant

import "encoding/json"

type DeviceConfig struct {
	Identifiers  []string `json:"identifiers"`
	Name         string   `json:"name"`
	Model        string   `json:"model,omitempty"`
	Manufacturer string   `json:"manufacturer,omitempty"`
}

type Config struct {
	DeviceClass       string        `json:"device_class"`
	StateClass        string        `json:"state_class"`
	Name              string        `json:"name"`
	StateTopic        string        `json:"state_topic,omitempty"`
	UnitOfMeasurement string        `json:"unit_of_measurement,omitempty"`
	ValueTemplate     string        `json:"value_template,omitempty"`
	UniqueId          string        `json:"unique_id,omitempty"`
	AvailabilityTopic string        `json:"availability_topic,omitempty"`
	Icon              string        `json:"icon,omitempty"`
	Device            *DeviceConfig `json:"device,omitempty"`
}

func (c *Config) Unmarshal(data []byte) (err error) {
	var parsed map[string]any
	err = json.Unmarshal(data, &parsed)
	if err != nil {
		return err
	}
	var ok bool
	if c.DeviceClass, ok = parsed["device_class"].(string); !ok {
		c.DeviceClass, _ = parsed["dev_cla"].(string)
	}
	if c.StateClass, ok = parsed["state_class"].(string); !ok {
		c.StateClass, _ = parsed["stat_cla"].(string)
	}
	c.Name, _ = parsed["name"].(string)
	if c.StateTopic, ok = parsed["state_topic"].(string); !ok {
		c.StateTopic, _ = parsed["stat_t"].(string)
	}
	if c.UnitOfMeasurement, ok = parsed["unit_of_measurement"].(string); !ok {
		c.UnitOfMeasurement, _ = parsed["unit_of_meas"].(string)
	}
	if c.ValueTemplate, ok = parsed["value_template"].(string); !ok {
		c.ValueTemplate, _ = parsed["val_tpl"].(string)
	}
	if c.UniqueId, ok = parsed["unique_id"].(string); !ok {
		c.UniqueId, _ = parsed["uniq_id"].(string)
	}
	if c.AvailabilityTopic, ok = parsed["availability_topic"].(string); !ok {
		c.AvailabilityTopic, _ = parsed["avty_t"].(string)
	}
	c.Icon, _ = parsed["icon"].(string)

	// device := parsed["device"].(map[string]any)
	// if device == nil {
	// 	device = parsed["dev"].(map[string]any)
	// }

	return nil
}
