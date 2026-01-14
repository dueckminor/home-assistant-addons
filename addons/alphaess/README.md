# AlphaESS Addon

This addon provides MQTT integration for AlphaESS solar inverter and battery systems.

## Features

- Real-time solar production monitoring via Modbus TCP
- Battery state of charge and power flow data
- Grid import/export statistics
- Home Assistant MQTT discovery integration
- Automatic sensor creation for all AlphaESS metrics

## Configuration

Configure the addon through the Home Assistant add-on interface:

- **mqtt_uri**: MQTT broker connection string (e.g., `tcp://core-mosquitto:1883`)
- **mqtt_user**: MQTT username (optional)
- **mqtt_password**: MQTT password (optional) 
- **alphaess_uri**: AlphaESS inverter Modbus TCP connection string (e.g., `tcp://192.168.1.100:502`)

## MQTT Topics

The addon publishes sensor data to MQTT topics under the `homeassistant/sensor/alphaess/` prefix with Home Assistant discovery enabled.

Example sensors:
- `homeassistant/sensor/alphaess/solar_production/state`
- `homeassistant/sensor/alphaess/battery_soc/state`  
- `homeassistant/sensor/alphaess/grid_active_power/state`

## Separation from MQTT Bridge

This addon was previously part of the MQTT Bridge addon but has been separated to:
- Allow independent development and updates
- Reduce complexity of the MQTT Bridge addon
- Provide focused configuration for AlphaESS integration

## Requirements

- AlphaESS inverter with Modbus TCP enabled
- Network connectivity between Home Assistant and the inverter
- MQTT broker (e.g., Mosquitto addon)