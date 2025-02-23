package main

import (
	"flag"
	"fmt"
	"os"
	"path"
	"time"

	"github.com/dueckminor/home-assistant-addons/go/services/influxdb"
	"gopkg.in/yaml.v3"
)

type InfluxCopyConfig struct {
	Source influxdb.InfluxDbConfig `yaml:"source"`
	Target influxdb.InfluxDbConfig `yaml:"target"`
}

var dataDir string

var theConfig InfluxCopyConfig

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

func main() {
	var client influxdb.Client
	var err error
	var series []influxdb.MetricSeries

	//measurement := "°C"
	measurement := "m³"
	targetTags := map[string]string{
		"device":       "homematic",
		"device_class": "gas",
		"domain":       "sensor",
		"entity_id":    "gas_energy_counter",
	}

	sourceTags := map[string]string{
		"entity_id": "count_gas_gas_energy_counter",
	}

	client, err = theConfig.Target.NewClient()
	if err != nil {
		panic(err)
	}

	series, err = client.GetMetricSeries(influxdb.MetricFilter{
		Measurement: measurement,
		Tags:        targetTags,
		Limit:       -1,
	})
	if err != nil {
		panic(err)
	}

	var after time.Time
	var before time.Time
	if len(series) > 0 && len(series[0].Values) > 0 {
		before = series[0].Values[0].Time
	}

	after, err = time.Parse(time.DateOnly, "2024-02-20")
	if err != nil {
		panic(err)
	}

	client, err = theConfig.Source.NewClient()
	if err != nil {
		panic(err)
	}
	series, err = client.GetMetricSeries(influxdb.MetricFilter{
		Measurement: measurement,
		Tags:        sourceTags,
		After:       after,
		Before:      before,
	})
	if err != nil {
		panic(err)
	}

	client, err = theConfig.Target.NewClient()
	if err != nil {
		panic(err)
	}

	for _, v := range series[0].Values {
		client.SendMetricAtTs(measurement, v.Value, targetTags, v.Time)
		fmt.Println(v.Time, v.Value)
	}

	time.Sleep(time.Second * 2)
}
