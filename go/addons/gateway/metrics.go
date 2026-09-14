package gateway

import (
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"sync"

	"github.com/dueckminor/home-assistant-addons/go/services/localmetrics"
	"github.com/dueckminor/home-assistant-addons/go/utils/network"
)

// MetricsCollector writes per-request metrics to local SQLite storage and
// manages an in-process geo-location cache.
type MetricsCollector struct {
	store      *localmetrics.Store
	geoCacheMu sync.RWMutex
	geoCache   map[string]*localmetrics.GeoLocation
}

func NewMetricsCollector(store *localmetrics.Store) *MetricsCollector {
	return &MetricsCollector{
		store:    store,
		geoCache: make(map[string]*localmetrics.GeoLocation),
	}
}

// RecordMetric persists a single request and triggers geo lookup if needed.
func (mc *MetricsCollector) RecordMetric(metric network.Metric) {
	clientIP := metric.ClientAddr
	if host, _, err := net.SplitHostPort(metric.ClientAddr); err == nil {
		clientIP = host
	}

	// Kick off a geo lookup asynchronously so it's warm for the next request.
	go mc.ensureGeoLocation(clientIP)

	durationMs := metric.Duration.Milliseconds()
	if err := mc.store.RecordRequest(
		metric.Timestamp,
		metric.Hostname,
		clientIP,
		metric.Method,
		metric.Path,
		metric.ResponseCode,
		durationMs,
		metric.Classification,
	); err != nil {
		fmt.Printf("metrics write error: %v\n", err)
	}
}

// ensureGeoLocation looks up geo data for the IP and caches it in the store.
func (mc *MetricsCollector) ensureGeoLocation(ipAddr string) {
	if localmetrics.IsPrivateIP(ipAddr) {
		geo := &localmetrics.GeoLocation{Country: "Local Network", CountryCode: "LC", City: "Local", Lat: 30, Lon: -40}
		mc.geoCacheMu.Lock()
		mc.geoCache[ipAddr] = geo
		mc.geoCacheMu.Unlock()
		_ = mc.store.SetGeoLocation(ipAddr, *geo)
		return
	}

	mc.geoCacheMu.RLock()
	_, cached := mc.geoCache[ipAddr]
	mc.geoCacheMu.RUnlock()
	if cached {
		return
	}

	if geo, found, err := mc.store.GetGeoLocation(ipAddr); err == nil && found {
		mc.geoCacheMu.Lock()
		mc.geoCache[ipAddr] = geo
		mc.geoCacheMu.Unlock()
		return
	}

	resp, err := http.Get(fmt.Sprintf("http://ip-api.com/json/%s?fields=status,country,countryCode,regionName,city,lat,lon,isp,org", ipAddr))
	if err != nil {
		fmt.Printf("geolocation lookup error for %s: %v\n", ipAddr, err)
		return
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
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil || apiResp.Status != "success" {
		return
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
	_ = mc.store.SetGeoLocation(ipAddr, *geo)
}
