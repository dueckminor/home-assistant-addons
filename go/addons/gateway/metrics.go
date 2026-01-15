package gateway

import (
	"encoding/json"
	"fmt"
	"hash/fnv"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/dueckminor/home-assistant-addons/go/services/influxdb"
	"github.com/dueckminor/home-assistant-addons/go/utils/network"
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
	Hostname      string
	Method        string
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

	// Remove the port from client address for key (safely)
	clientIP := metric.ClientAddr
	if host, _, err := net.SplitHostPort(metric.ClientAddr); err == nil {
		clientIP = host
	}

	hostname := metric.Hostname
	if hostname == "" {
		hostname = "NONE"
	}
	method := metric.Method
	if method == "" {
		method = "NONE"
	}

	key = fmt.Sprintf("%s/%s/%s", clientIP, hostname, method)

	metrics, exists := mc.routes[key]
	if !exists {
		metrics = &RouteMetrics{
			Hostname:    hostname,
			Method:      method,
			ClientAddr:  clientIP,
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

	for _, metrics := range snapshot {
		// Parse route key: "clientaddr:hostname:method:path"

		// Resolve geolocation for this client IP (async, won't block requests)
		if metrics.GeoLocation == nil {
			metrics.GeoLocation = mc.getGeoLocation(metrics.ClientAddr)
		}

		// Create optimized tags (low cardinality)
		tags := map[string]string{
			"hostname": metrics.Hostname,
			"method":   metrics.Method,
		}

		// Create comprehensive fields (numeric and string data)
		fields := map[string]interface{}{
			"request_count": float64(metrics.RequestCount),
			"error_count":   float64(metrics.ErrorCount),
			"client_ip":     metrics.ClientAddr,
		}

		// Add response time fields
		if metrics.RequestCount > 0 {
			fields["response_time_avg"] = float64(metrics.TotalDuration.Milliseconds()) / float64(metrics.RequestCount)
		}
		if metrics.MinDuration > 0 {
			fields["response_time_min"] = float64(metrics.MinDuration.Milliseconds())
		}
		if metrics.MaxDuration > 0 {
			fields["response_time_max"] = float64(metrics.MaxDuration.Milliseconds())
		}

		// Add geolocation fields (numeric and string)
		if metrics.GeoLocation != nil {
			fields["latitude"] = metrics.GeoLocation.Lat
			fields["longitude"] = metrics.GeoLocation.Lon
			fields["country_name"] = metrics.GeoLocation.Country
			fields["city_name"] = metrics.GeoLocation.City
		}

		// Add individual status code counts as fields
		var status2xx, status3xx, status4xx, status5xx, statusSpecial float64
		for statusCode, count := range metrics.StatusCodes {
			fieldName := fmt.Sprintf("status_%d", statusCode)
			fields[fieldName] = float64(count)

			// Group status codes by category
			switch {
			case statusCode >= 200 && statusCode < 300:
				status2xx += float64(count)
			case statusCode >= 300 && statusCode < 400:
				status3xx += float64(count)
			case statusCode >= 400 && statusCode < 500:
				status4xx += float64(count)
			case statusCode >= 500 && statusCode < 600:
				status5xx += float64(count)
			case statusCode == 666 || statusCode == 667:
				statusSpecial += float64(count)
			}
		}

		// Add grouped status code fields
		if status2xx > 0 {
			fields["status_2xx"] = status2xx
		}
		if status3xx > 0 {
			fields["status_3xx"] = status3xx
		}
		if status4xx > 0 {
			fields["status_4xx"] = status4xx
		}
		if status5xx > 0 {
			fields["status_5xx"] = status5xx
		}
		if statusSpecial > 0 {
			fields["status_special"] = statusSpecial
		}

		// Send single consolidated metric
		if err := mc.influxDB.SendMetricWithFieldsAtTs("gateway_metrics", fields, tags, now); err != nil {
			fmt.Printf("Failed to send gateway metrics: %v\n", err)
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

// simpleHash creates a simple numeric hash of a string for privacy-preserving IP tracking
func simpleHash(s string) uint32 {
	h := fnv.New32a()
	h.Write([]byte(s))
	return h.Sum32()
}
