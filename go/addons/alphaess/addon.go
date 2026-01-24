package alphaess

import (
	"fmt"
	"path"
	"time"

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

type addon struct {
	config  AlphaEssConfig
	scanner alphaess.Scanner
	db      sqlite.Database
}

type MeasurementFilter = alphaess.MeasurementFilter
type Measurement = alphaess.Measurement

type Addon interface {
	addons.Addon
	GetMeasurements(filter MeasurementFilter) []Measurement
	GetGaps(filter MeasurementFilter) []Measurement
}

func NewAddon(config AlphaEssAddonConfig) Addon {
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

	if config.MqttURI != "" {
		mqttBroker := mqtt.NewBroker(config.MqttURI, config.MqttUser, config.MqttPassword)
		mqttConn, err := mqttBroker.Dial(mqttClientId, "")
		if err != nil {
			panic(err)
		}
		defer mqttConn.Close()

		automation.GetRegistry().EnableMqtt(mqttBroker)
		automation.GetRegistry().EnableHomeAssistant()
	}

	// Start AlphaESS integration
	scanner, err := alphaess.Run(config.AlphaEssUri)
	if err != nil {
		panic(err)
	}

	return &addon{
		config:  config.AlphaEssConfig,
		scanner: scanner,
		db:      db,
	}
}

func (a *addon) Endpoints() addons.Endpoints {
	return NewEndpoints(a)
}

func makeQueryAndParams(column string, statisticId string, filter MeasurementFilter, sibling int) (string, []any) {
	query := "SELECT start_ts, " + column + " FROM statistics\n"
	query += "WHERE metadata_id = (SELECT id FROM statistics_meta WHERE statistic_id=?)\n"
	params := []any{statisticId}

	switch sibling {
	case -1:
		query += "AND start_ts <= ?\n"
		params = append(params, float64(filter.After.Unix()))
		query += "ORDER BY start_ts DESC LIMIT 1\n"
	case 0:
		if !filter.Before.IsZero() {
			query += "AND start_ts < ?\n"
			params = append(params, float64(filter.Before.Unix()))
		}
		if !filter.NotBefore.IsZero() {
			query += "AND start_ts >= ?\n"
			params = append(params, float64(filter.NotBefore.Unix()))
		}
		if !filter.After.IsZero() {
			query += "AND start_ts > ?\n"
			params = append(params, float64(filter.After.Unix()))
		}
		if !filter.NotAfter.IsZero() {
			query += "AND start_ts <= ?\n"
			params = append(params, float64(filter.NotAfter.Unix()))
		}
		query += "ORDER BY start_ts ASC\n"
	case 1:
		query += "AND start_ts >= ?\n"
		params = append(params, float64(filter.Before.Unix()))
		query += "ORDER BY start_ts ASC LIMIT 1\n"
	}
	return query, params
}

func (a *addon) getMeasurementValuesFromDB(column string, statisticId string, filter MeasurementFilter, sibling int) (values []alphaess.MeasurementValue) {
	query, params := makeQueryAndParams(column, statisticId, filter, sibling)

	rows, err := a.db.Query(query, params...)
	if err != nil {
		fmt.Println("Error querying measurements:", err)
		return []alphaess.MeasurementValue{}
	}

	for rows.Next() {
		var ts float64
		var state float64
		err = rows.Scan(&ts, &state)
		if err != nil {
			return []alphaess.MeasurementValue{}
		}
		timestamp := time.Unix(int64(ts), 0)
		values = append(values, alphaess.MeasurementValue{
			Time:  timestamp.UTC(),
			Value: state,
		})
	}

	return values
}

func (a *addon) getMeasurementsFromDB(filter MeasurementFilter) []Measurement {
	result := a.scanner.GetMeasurementInfos(filter)

	for i := range result {
		column := "state"
		statisticId := "sensor.alpha_ess_" + result[i].Name

		if result[i].Name == "battery_soc" {
			column = "mean"
		}

		if filter.Previous || filter.Siblings {
			result[i].Values = a.getMeasurementValuesFromDB(column, statisticId, filter, -1)
		}
		result[i].Values = append(result[i].Values, a.getMeasurementValuesFromDB(column, statisticId, filter, 0)...)
		if filter.Siblings {
			result[i].Values = append(result[i].Values, a.getMeasurementValuesFromDB(column, statisticId, filter, 1)...)
		}
	}

	return result
}

func (a *addon) GetMeasurements(filter MeasurementFilter) []Measurement {
	if filter.TimeSpecified() {
		return a.getMeasurementsFromDB(filter)
	}
	return a.scanner.GetMeasurements(filter)
}

func (a *addon) GetGaps(filter MeasurementFilter) []Measurement {
	result := a.getMeasurementsFromDB(filter)

	for i := range result {
		result[i].Values = a.findGaps(result[i].Values)
	}

	return result
}

func (a *addon) findGaps(values []alphaess.MeasurementValue) []alphaess.MeasurementValue {
	// Define a gap as a period longer than one hour without measurements
	const gapThreshold = 1*time.Hour + 30*time.Minute

	var gaps []alphaess.MeasurementValue
	if len(values) < 2 {
		return gaps
	}

	for i := 1; i < len(values); i++ {
		prev := values[i-1]
		curr := values[i]

		if curr.Time.Sub(prev.Time) > gapThreshold {
			// add gap for each full hour between prev and curr
			gapStart := prev.Time.Add(1 * time.Hour)
			// set minute, second, nanosecond to zero
			gapStart = time.Date(gapStart.Year(), gapStart.Month(), gapStart.Day(), gapStart.Hour(), 0, 0, 0, gapStart.Location())
			for gapStart.Before(curr.Time) {
				gaps = append(gaps, alphaess.MeasurementValue{
					Time:  gapStart,
					Value: 0,
				})
				gapStart = gapStart.Add(1 * time.Hour)
			}
		}
	}

	return gaps
}
