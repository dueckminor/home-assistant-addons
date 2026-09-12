package gateway

import (
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/dueckminor/home-assistant-addons/go/services/localmetrics"
	"github.com/dueckminor/home-assistant-addons/go/utils/network"
)

// RouteMetrics stores metrics for a specific route and client during the aggregation window
type RouteMetrics struct {
	ClientAddr    string
	Hostname      string
	Method        string
	GeoLocation   *localmetrics.GeoLocation
	RequestCount  int64
	TotalDuration time.Duration
	MinDuration   time.Duration
	MaxDuration   time.Duration
	ErrorCount    int64
}

// MetricsCollector aggregates HTTP metrics and writes them to local SQLite storage periodically
type MetricsCollector struct {
	mu         sync.Mutex
	routes     map[string]*RouteMetrics
	store      *localmetrics.Store
	interval   time.Duration
	stopChan   chan struct{}
	wg         sync.WaitGroup
	geoCache   map[string]*localmetrics.GeoLocation
	geoCacheMu sync.RWMutex
}

// NewMetricsCollector creates a new metrics collector
func NewMetricsCollector(store *localmetrics.Store, interval time.Duration) *MetricsCollector {
	return &MetricsCollector{
		routes:   make(map[string]*RouteMetrics),
		store:    store,
		interval: interval,
		stopChan: make(chan struct{}),
		geoCache: make(map[string]*localmetrics.GeoLocation),
	}
}

// RecordMetric records metrics from a network.Metric
func (mc *MetricsCollector) RecordMetric(metric network.Metric) {
	mc.mu.Lock()
	defer mc.mu.Unlock()

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

	key := fmt.Sprintf("%s/%s/%s", clientIP, hostname, method)

	metrics, exists := mc.routes[key]
	if !exists {
		metrics = &RouteMetrics{
			Hostname:    hostname,
			Method:      method,
			ClientAddr:  clientIP,
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
}

func (mc *MetricsCollector) reportLoop() {
	defer mc.wg.Done()

	ticker := time.NewTicker(mc.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			mc.sendMetrics()
		case <-mc.stopChan:
			mc.sendMetrics()
			if err := mc.store.Cleanup(90); err != nil {
				fmt.Printf("metrics cleanup error: %v\n", err)
			}
			return
		}
	}
}

func (mc *MetricsCollector) sendMetrics() {
	mc.mu.Lock()
	snapshot := mc.routes
	mc.routes = make(map[string]*RouteMetrics)
	mc.mu.Unlock()

	if len(snapshot) == 0 {
		return
	}

	bucketStart := time.Now().Truncate(mc.interval)

	for _, metrics := range snapshot {
		if metrics.GeoLocation == nil {
			metrics.GeoLocation = mc.getGeoLocation(metrics.ClientAddr)
		}

		if metrics.GeoLocation != nil {
			if err := mc.store.SetGeoLocation(metrics.ClientAddr, *metrics.GeoLocation); err != nil {
				fmt.Printf("geo cache write error: %v\n", err)
			}
		}

		var avgMs int64
		if metrics.RequestCount > 0 {
			avgMs = metrics.TotalDuration.Milliseconds() / metrics.RequestCount
		}

		if err := mc.store.RecordBatch(bucketStart, metrics.Hostname, metrics.ClientAddr, metrics.Method, localmetrics.RouteMetricsSnapshot{
			RequestCount:  metrics.RequestCount,
			ErrorCount:    metrics.ErrorCount,
			DurationAvgMs: avgMs,
			DurationMinMs: metrics.MinDuration.Milliseconds(),
			DurationMaxMs: metrics.MaxDuration.Milliseconds(),
		}); err != nil {
			fmt.Printf("metrics write error: %v\n", err)
		}
	}
}

func (mc *MetricsCollector) getGeoLocation(ipAddr string) *localmetrics.GeoLocation {
	if localmetrics.IsPrivateIP(ipAddr) {
		return &localmetrics.GeoLocation{Country: "Local", CountryCode: "LC", City: "Localhost"}
	}

	// Check in-memory cache
	mc.geoCacheMu.RLock()
	if cached, exists := mc.geoCache[ipAddr]; exists {
		mc.geoCacheMu.RUnlock()
		return cached
	}
	mc.geoCacheMu.RUnlock()

	// Check persisted SQLite cache
	if geo, found, err := mc.store.GetGeoLocation(ipAddr); err == nil && found {
		mc.geoCacheMu.Lock()
		mc.geoCache[ipAddr] = geo
		mc.geoCacheMu.Unlock()
		return geo
	}

	// Call ip-api.com
	resp, err := http.Get(fmt.Sprintf("http://ip-api.com/json/%s?fields=status,country,countryCode,regionName,city,lat,lon,isp,org", ipAddr))
	if err != nil {
		fmt.Printf("geolocation lookup error for %s: %v\n", ipAddr, err)
		return nil
	}
	defer resp.Body.Close()

	var apiResp struct {
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
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		fmt.Printf("geolocation decode error for %s: %v\n", ipAddr, err)
		return nil
	}
	if apiResp.Status != "success" {
		return nil
	}

	geo := &localmetrics.GeoLocation{
		Country:     apiResp.Country,
		CountryCode: apiResp.CountryCode,
		City:        apiResp.City,
		Region:      apiResp.Region,
		Lat:         apiResp.Lat,
		Lon:         apiResp.Lon,
		ISP:         apiResp.ISP,
		Org:         apiResp.Org,
	}

	mc.geoCacheMu.Lock()
	mc.geoCache[ipAddr] = geo
	mc.geoCacheMu.Unlock()

	return geo
}
