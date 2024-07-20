package influxdb

import (
	"fmt"
	"time"

	influxdb1 "github.com/influxdata/influxdb/client/v2"
)

type Client interface {
	Close() error
	Flush()
	SendMetric(name string, value float64)
	SendMetricAtTs(name string, value float64, ts time.Time)
}

func Foo(uri, user, password string) (err error) {
	client, err := influxdb1.NewHTTPClient(influxdb1.HTTPConfig{
		Addr:     uri,
		Username: user,
		Password: password,
	})
	if err != nil {
		return err
	}
	q := influxdb1.NewQuery(
		`SELECT time,value,device_class,domain,entity_id,friendly_name,source FROM "Wh" WHERE  "entity_id"='from_grid' ORDER BY time`, `mypi`, ``)
	resp, err := client.Query(q)
	if err != nil {
		return err
	}

	var thisTime time.Time
	var lastTime time.Time

	for _, result := range resp.Results {
		for _, rows := range result.Series {
			for i, row := range rows.Values {
				if value, ok := row[0].(string); ok {

					parsedTime, err := time.Parse(time.RFC3339, value)
					if err == nil {
						lastTime = thisTime
						thisTime = parsedTime
					}
				}

				delta := thisTime.Sub(lastTime)
				if i == 0 {
					fmt.Println(row[1], thisTime)
				} else if delta > 2*time.Minute {
					fmt.Println(row[1], thisTime, delta)
				}
			}
		}
	}

	// client.Write()

	// influxdb1.NewBatchPoints()

	// influxdb1.NewPoint("Wh",map[string]string{
	// 	"device_class", "energy",
	// 	"domain", "sensor",
	// 	"device", "alphaess",
	// 	"source", "mypi",
	// 	"entity_id", name)
	// })
	return nil
}
