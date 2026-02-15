package network

import (
	"time"

	"github.com/gin-gonic/gin"
)

type Metric struct {
	Timestamp    time.Time
	ClientAddr   string
	Duration     time.Duration
	Hostname     string
	Method       string
	Path         string
	ResponseCode int
}

type MetricCallback func(metric Metric)

func MetricMiddleware(callback MetricCallback) func(c *gin.Context) {
	return func(c *gin.Context) {
		metric := Metric{
			Timestamp:  time.Now(),
			ClientAddr: c.ClientIP(),
		}
		defer func() {
			metric.Duration = time.Since(metric.Timestamp)
			metric.Hostname = c.Request.Host
			metric.Method = c.Request.Method
			metric.Path = c.Request.URL.Path
			metric.ResponseCode = c.Writer.Status()
			callback(metric)
		}()
		c.Next()
	}
}
