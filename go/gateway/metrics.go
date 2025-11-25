package gateway

import (
	"fmt"
	"sync"
	"time"

	"github.com/dueckminor/home-assistant-addons/go/services/influxdb"
	"github.com/gin-gonic/gin"
)

// RouteMetrics stores aggregated metrics for a specific route
type RouteMetrics struct {
	RequestCount  int64
	TotalDuration time.Duration
	MinDuration   time.Duration
	MaxDuration   time.Duration
	ErrorCount    int64
	StatusCodes   map[int]int64
}

// MetricsCollector aggregates HTTP metrics and sends them to InfluxDB periodically
type MetricsCollector struct {
	mu       sync.Mutex
	routes   map[string]*RouteMetrics
	influxDB influxdb.Client
	interval time.Duration
	stopChan chan struct{}
	wg       sync.WaitGroup
}

// NewMetricsCollector creates a new metrics collector
func NewMetricsCollector(influxClient influxdb.Client, interval time.Duration) *MetricsCollector {
	return &MetricsCollector{
		routes:   make(map[string]*RouteMetrics),
		influxDB: influxClient,
		interval: interval,
		stopChan: make(chan struct{}),
	}
}

// RecordRequest records metrics for a single HTTP request
func (mc *MetricsCollector) RecordRequest(route string, method string, statusCode int, duration time.Duration) {
	mc.mu.Lock()
	defer mc.mu.Unlock()

	key := fmt.Sprintf("%s:%s", method, route)

	metrics, exists := mc.routes[key]
	if !exists {
		metrics = &RouteMetrics{
			StatusCodes: make(map[int]int64),
			MinDuration: duration,
		}
		mc.routes[key] = metrics
	}

	metrics.RequestCount++
	metrics.TotalDuration += duration

	if duration < metrics.MinDuration {
		metrics.MinDuration = duration
	}
	if duration > metrics.MaxDuration {
		metrics.MaxDuration = duration
	}

	if statusCode >= 400 {
		metrics.ErrorCount++
	}

	metrics.StatusCodes[statusCode]++
}

// Start begins the periodic metrics reporting
func (mc *MetricsCollector) Start() {
	mc.wg.Add(1)
	go mc.reportLoop()
}

// Stop stops the metrics collector
func (mc *MetricsCollector) Stop() {
	close(mc.stopChan)
	mc.wg.Wait()
	if mc.influxDB != nil {
		mc.influxDB.Close()
	}
}

// reportLoop sends metrics to InfluxDB at regular intervals
func (mc *MetricsCollector) reportLoop() {
	defer mc.wg.Done()

	ticker := time.NewTicker(mc.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			mc.sendMetrics()
		case <-mc.stopChan:
			// Send final metrics before stopping
			mc.sendMetrics()
			return
		}
	}
}

// sendMetrics sends all collected metrics to InfluxDB and resets counters
func (mc *MetricsCollector) sendMetrics() {
	mc.mu.Lock()

	// Snapshot current metrics and reset
	snapshot := mc.routes
	mc.routes = make(map[string]*RouteMetrics)

	mc.mu.Unlock()

	if mc.influxDB == nil || len(snapshot) == 0 {
		return
	}

	now := time.Now()

	for routeKey, metrics := range snapshot {
		tags := map[string]string{
			"route": routeKey,
		}

		// Send request count
		if err := mc.influxDB.SendMetricAtTs("gateway_requests", float64(metrics.RequestCount), tags, now); err != nil {
			fmt.Printf("Failed to send request count metric: %v\n", err)
		}

		// Send average response time (in milliseconds)
		if metrics.RequestCount > 0 {
			avgMs := float64(metrics.TotalDuration.Milliseconds()) / float64(metrics.RequestCount)
			if err := mc.influxDB.SendMetricAtTs("gateway_response_time_avg", avgMs, tags, now); err != nil {
				fmt.Printf("Failed to send avg response time metric: %v\n", err)
			}
		}

		// Send min response time (in milliseconds)
		if metrics.MinDuration > 0 {
			if err := mc.influxDB.SendMetricAtTs("gateway_response_time_min", float64(metrics.MinDuration.Milliseconds()), tags, now); err != nil {
				fmt.Printf("Failed to send min response time metric: %v\n", err)
			}
		}

		// Send max response time (in milliseconds)
		if metrics.MaxDuration > 0 {
			if err := mc.influxDB.SendMetricAtTs("gateway_response_time_max", float64(metrics.MaxDuration.Milliseconds()), tags, now); err != nil {
				fmt.Printf("Failed to send max response time metric: %v\n", err)
			}
		}

		// Send error count
		if metrics.ErrorCount > 0 {
			if err := mc.influxDB.SendMetricAtTs("gateway_errors", float64(metrics.ErrorCount), tags, now); err != nil {
				fmt.Printf("Failed to send error count metric: %v\n", err)
			}
		}

		// Send status code distribution
		for statusCode, count := range metrics.StatusCodes {
			statusTags := map[string]string{
				"route":  routeKey,
				"status": fmt.Sprintf("%d", statusCode),
			}
			if err := mc.influxDB.SendMetricAtTs("gateway_status_codes", float64(count), statusTags, now); err != nil {
				fmt.Printf("Failed to send status code metric: %v\n", err)
			}
		}
	}
}

// MetricsMiddleware returns a Gin middleware that records HTTP metrics
func (mc *MetricsCollector) MetricsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		if mc == nil {
			c.Next()
			return
		}

		start := time.Now()

		// Process request
		c.Next()

		// Record metrics
		duration := time.Since(start)
		route := c.FullPath()
		if route == "" {
			route = c.Request.URL.Path
		}
		method := c.Request.Method
		statusCode := c.Writer.Status()

		mc.RecordRequest(route, method, statusCode, duration)
	}
}
