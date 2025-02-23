package homematic

import (
	"fmt"
	"time"

	"github.com/dueckminor/home-assistant-addons/go/services/automation"
)

type HomematicConfig struct {
	HomematicUri      string            `yaml:"homematic_uri"`
	HomematicUser     string            `yaml:"homematic_user"`
	HomematicPassword string            `yaml:"homematic_password"`
	HomematicDevices  []HomematicDevice `yaml:"homematic_devices"`
}

type HomematicDevice struct {
	Address    string `yaml:"address"`
	DeviceType string `yaml:"device_type"`
}

func StartMqttBridge(config HomematicConfig) (err error) {
	registry := automation.GetRegistry()
	node := registry.CreateNode("homematic")
	sensorGas := node.CreateSensor(automation.MakeSensorTemplate("gas_energy_counter").
		SetIcon(automation.Icon_MeterGas).
		SetUnit(automation.Unit_m3).
		SetStateClass(automation.StateClass_TotalIncreasing).
		SetDeviceClass(automation.DeviceClass_Gas).
		SetPrecision(2))

	ccuc, err := NewCcuClient(config.HomematicUri, config.HomematicUser, config.HomematicPassword)
	if err != nil {
		return err
	}

	ccuc.SetCallback(func(dev Device, valueKey string, value interface{}) {
		if valueKey == "GAS_ENERGY_COUNTER" {
			sensorGas.SetState(value)
		}
		fmt.Println(dev.Name(), dev.Type(), dev.Address(), valueKey, value)
	})

	var temperatureDevices []Device

	devices, _ := ccuc.GetDevices()
	for _, device := range devices {
		fmt.Println(device.Name(), device.Type(), device.Address())
		if device.Type() == "HmIP-WTH-2" {
			subdevice, _ := device.GetSubDevice("HEATING_CLIMATECONTROL_TRANSCEIVER")
			if subdevice != nil {
				temperatureDevices = append(temperatureDevices, subdevice)
			}
		}
	}

	go func() {
		for {
			for _, device := range temperatureDevices {
				device.SetValue("WINDOW_STATE", 2)
			}
			time.Sleep(time.Minute * 5)
		}
	}()

	return ccuc.StartCallbackHandler()
}
