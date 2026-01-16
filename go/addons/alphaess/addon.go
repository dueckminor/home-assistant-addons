package alphaess

import (
	"fmt"
	"path"

	"github.com/dueckminor/home-assistant-addons/go/addons"
	"github.com/dueckminor/home-assistant-addons/go/services/alphaess"
	"github.com/dueckminor/home-assistant-addons/go/services/automation"
	"github.com/dueckminor/home-assistant-addons/go/services/mqtt"
	"github.com/dueckminor/home-assistant-addons/go/services/sqlite"
	"github.com/dueckminor/home-assistant-addons/go/utils/crypto/rand"
)

type MqttConfig struct {
	MqttURI      string `yaml:"mqtt_uri"`
	MqttUser     string `yaml:"mqtt_user"`
	MqttPassword string `yaml:"mqtt_password"`
}

type AlphaEssConfig struct {
	MqttConfig  `yaml:",inline"`
	AlphaEssUri string `yaml:"alphaess_uri"`
}

type AlphaEssAddonConfig struct {
	AlphaEssConfig
	HomeAssistantConfigDir string
}

type Addon struct {
	config AlphaEssConfig
	db     sqlite.Database
}

func NewAddon(config AlphaEssAddonConfig) addons.Addon {
	id, err := rand.GetString(10)
	if err != nil {
		panic(err)
	}

	mqttClientId := "alphaess-" + id

	fmt.Println("MQTT URI:", config.MqttURI)
	fmt.Println("MQTT Client ID:", mqttClientId)
	fmt.Println("AlphaESS URI:", config.AlphaEssUri)

	dbFile := path.Join(config.HomeAssistantConfigDir, "home-assistant_v2.db")
	db, err := sqlite.OpenDatabase(dbFile)
	if err != nil {
		panic(err)
	}

	if config.AlphaEssUri == "" {
		fmt.Println("AlphaESS URI not configured, exiting...")
		return nil
	}

	mqttBroker := mqtt.NewBroker(config.MqttURI, config.MqttUser, config.MqttPassword)
	mqttConn, err := mqttBroker.Dial(mqttClientId, "")
	if err != nil {
		panic(err)
	}
	defer mqttConn.Close()

	automation.GetRegistry().EnableMqtt(mqttBroker)
	automation.GetRegistry().EnableHomeAssistant()

	// Start AlphaESS integration
	alphaess.Run(config.AlphaEssUri)

	return &Addon{
		config: config.AlphaEssConfig,
		db:     db,
	}
}

func (a *Addon) Endpoints() addons.Endpoints {
	return NewEndpoints(a)
}
