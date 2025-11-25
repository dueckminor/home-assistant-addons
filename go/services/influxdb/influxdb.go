package influxdb

import (
	"fmt"
	"time"

	_ "github.com/influxdata/influxdb1-client" // this is important because of the bug in go mod
	influxdb1 "github.com/influxdata/influxdb1-client/v2"
)

type Client interface {
	Close() error
	Flush()
	SendMetric(measurement string, value float64, tags map[string]string) error
	SendMetricAtTs(measurement string, value float64, tags map[string]string, ts time.Time) error
}

type client struct {
	client influxdb1.Client
	config influxdb1.BatchPointsConfig
}

func (c *client) Close() error {
	return c.client.Close()
}

func (c *client) Flush() {

}

func (c *client) SendMetric(measurement string, value float64, tags map[string]string) error {
	return c.SendMetricAtTs(measurement, value, tags, time.Now().UTC())
}

func (c *client) SendMetricAtTs(measurement string, value float64, tags map[string]string, ts time.Time) error {
	points, err := influxdb1.NewBatchPoints(c.config)
	if err != nil {
		return fmt.Errorf("failed to create batch points: %w", err)
	}
	point, err := influxdb1.NewPoint(measurement, tags, map[string]any{"value": value}, ts)
	if err != nil {
		return fmt.Errorf("failed to create point: %w", err)
	}

	points.AddPoint(point)

	err = c.client.Write(points)
	if err != nil {
		return fmt.Errorf("failed to write to InfluxDB: %w", err)
	}

	return nil
}

func NewClient(uri, database, user, password string) (c Client, err error) {
	client := &client{}
	client.config.Database = database
	client.client, err = influxdb1.NewHTTPClient(influxdb1.HTTPConfig{
		Addr:     uri,
		Username: user,
		Password: password,
	})
	if err != nil {
		return nil, err
	}
	return client, nil
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

	return nil
}
