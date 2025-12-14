package gateway

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/dueckminor/home-assistant-addons/go/network"
	"github.com/dueckminor/home-assistant-addons/go/services/influxdb"
)

// GeoLocation stores geographical information for an IP address
type GeoLocation struct {
	Country     string  `json:"country"`
	CountryCode string  `json:"countryCode"`
	City        string  `json:"city"`
	Region      string  `json:"regionName"`
	Lat         float64 `json:"lat"`
	Lon         float64 `json:"lon"`
	ISP         string  `json:"isp"`
	Org         string  `json:"org"`
}

// RouteMetrics stores metrics for a specific route and client
type RouteMetrics struct {
	ClientAddr    string
	GeoLocation   *GeoLocation
	RequestCount  int64
	TotalDuration time.Duration
	MinDuration   time.Duration
	MaxDuration   time.Duration
	ErrorCount    int64
	StatusCodes   map[int]int64
}

// MetricsCollector aggregates HTTP metrics and sends them to InfluxDB periodically
type MetricsCollector struct {
	mu         sync.Mutex
	routes     map[string]*RouteMetrics
	influxDB   influxdb.Client
	interval   time.Duration
	stopChan   chan struct{}
	wg         sync.WaitGroup
	geoCache   map[string]*GeoLocation
	geoCacheMu sync.RWMutex
}

// NewMetricsCollector creates a new metrics collector
func NewMetricsCollector(influxClient influxdb.Client, interval time.Duration) *MetricsCollector {
	return &MetricsCollector{
		routes:   make(map[string]*RouteMetrics),
		influxDB: influxClient,
		interval: interval,
		stopChan: make(chan struct{}),
		geoCache: make(map[string]*GeoLocation),
	}
}

// RecordMetric records metrics from a network.Metric
func (mc *MetricsCollector) RecordMetric(metric network.Metric) {
	mc.mu.Lock()
	defer mc.mu.Unlock()

	// Create key from client address, hostname, method, and path for individual client tracking
	// Special cases:
	// - ResponseCode 666: Unknown hostname (port scan attack) - no method/path
	// - ResponseCode 667: TLS handshake failure - no method/path
	var key string
	if metric.ResponseCode == 666 {
		key = fmt.Sprintf("%s:%s:UNKNOWN:unknown_host", metric.ClientAddr, metric.Hostname)
	} else if metric.ResponseCode == 667 {
		key = fmt.Sprintf("%s:%s:TLS:handshake_failed", metric.ClientAddr, metric.Hostname)
	} else {
		key = fmt.Sprintf("%s:%s:%s:%s", metric.ClientAddr, metric.Hostname, metric.Method, metric.Path)
	}

	metrics, exists := mc.routes[key]
	if !exists {
		metrics = &RouteMetrics{
			ClientAddr:  metric.ClientAddr,
			GeoLocation: nil, // Will be resolved during sendMetrics
			StatusCodes: make(map[int]int64),
			MinDuration: metric.Duration,
		}
		mc.routes[key] = metrics
	}

	metrics.RequestCount++
	metrics.TotalDuration += metric.Duration

	if metric.Duration < metrics.MinDuration || metrics.MinDuration == 0 {
		metrics.MinDuration = metric.Duration
	}
	if metric.Duration > metrics.MaxDuration {
		metrics.MaxDuration = metric.Duration
	}

	if metric.ResponseCode >= 400 || metric.ResponseCode == 666 || metric.ResponseCode == 667 {
		metrics.ErrorCount++
	}

	metrics.StatusCodes[metric.ResponseCode]++
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
		// Parse the route key to extract client, hostname, method, and path
		// Format: "clientaddr:hostname:method:path"
		tags := map[string]string{
			"route":       routeKey,
			"client_addr": metrics.ClientAddr,
		}

		// Resolve geolocation for this client IP (async, won't block requests)
		if metrics.GeoLocation == nil {
			metrics.GeoLocation = mc.getGeoLocation(metrics.ClientAddr)
		}

		// Add geolocation tags if available
		if metrics.GeoLocation != nil {
			tags["country"] = metrics.GeoLocation.Country
			tags["country_code"] = metrics.GeoLocation.CountryCode
			tags["city"] = metrics.GeoLocation.City
			tags["region"] = metrics.GeoLocation.Region
			tags["isp"] = metrics.GeoLocation.ISP
			tags["org"] = metrics.GeoLocation.Org
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
				"route":       routeKey,
				"client_addr": metrics.ClientAddr,
				"status":      fmt.Sprintf("%d", statusCode),
			}
			// Add geolocation tags to status codes as well
			if metrics.GeoLocation != nil {
				statusTags["country"] = metrics.GeoLocation.Country
				statusTags["country_code"] = metrics.GeoLocation.CountryCode
				statusTags["city"] = metrics.GeoLocation.City
				statusTags["region"] = metrics.GeoLocation.Region
				statusTags["isp"] = metrics.GeoLocation.ISP
				statusTags["org"] = metrics.GeoLocation.Org
			}
			if err := mc.influxDB.SendMetricAtTs("gateway_status_codes", float64(count), statusTags, now); err != nil {
				fmt.Printf("Failed to send status code metric: %v\n", err)
			}
		}
	}
}

// getGeoLocation retrieves geolocation data for an IP address
func (mc *MetricsCollector) getGeoLocation(ipAddr string) *GeoLocation {
	// Skip localhost and private IPs
	if ipAddr == "127.0.0.1" || ipAddr == "::1" || ipAddr == "localhost" {
		return &GeoLocation{
			Country:     "Local",
			CountryCode: "LC",
			City:        "Localhost",
			Region:      "Local",
			ISP:         "Local",
			Org:         "Local",
		}
	}

	// Check cache first
	mc.geoCacheMu.RLock()
	if cached, exists := mc.geoCache[ipAddr]; exists {
		mc.geoCacheMu.RUnlock()
		return cached
	}
	mc.geoCacheMu.RUnlock()

	// Use free ip-api.com service (100 requests per minute limit)
	resp, err := http.Get(fmt.Sprintf("http://ip-api.com/json/%s?fields=status,country,countryCode,regionName,city,lat,lon,isp,org", ipAddr))
	if err != nil {
		fmt.Printf("Failed to get geolocation for %s: %v\n", ipAddr, err)
		return nil
	}
	defer resp.Body.Close()

	var apiResponse struct {
		Status      string  `json:"status"`
		Country     string  `json:"country"`
		CountryCode string  `json:"countryCode"`
		Region      string  `json:"regionName"`
		City        string  `json:"city"`
		Lat         float64 `json:"lat"`
		Lon         float64 `json:"lon"`
		ISP         string  `json:"isp"`
		Org         string  `json:"org"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&apiResponse); err != nil {
		fmt.Printf("Failed to decode geolocation response for %s: %v\n", ipAddr, err)
		return nil
	}

	if apiResponse.Status != "success" {
		return nil
	}

	geoLocation := &GeoLocation{
		Country:     apiResponse.Country,
		CountryCode: apiResponse.CountryCode,
		City:        apiResponse.City,
		Region:      apiResponse.Region,
		Lat:         apiResponse.Lat,
		Lon:         apiResponse.Lon,
		ISP:         apiResponse.ISP,
		Org:         apiResponse.Org,
	}

	// Cache the result
	mc.geoCacheMu.Lock()
	mc.geoCache[ipAddr] = geoLocation
	mc.geoCacheMu.Unlock()

	return geoLocation
}
