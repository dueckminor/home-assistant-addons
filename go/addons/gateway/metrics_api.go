package gateway

import (
	"time"

	"github.com/dueckminor/home-assistant-addons/go/services/localmetrics"
	"github.com/gin-gonic/gin"
)

func (ep *Endpoints) GET_MetricsConfig(c *gin.Context) {
	key := ep.Gateway.config.Metrics.CartoApiKey
	styleURL := "https://basemaps.cartocdn.com/gl/voyager-gl-style/style.json"
	if key != "" {
		styleURL += "?api_key=" + key
	}
	c.JSON(200, gin.H{
		"style_url": styleURL,
	})
}

func (ep *Endpoints) GET_MetricsHostnames(c *gin.Context) {
	if ep.Gateway.metricsStore == nil {
		c.JSON(503, gin.H{"error": "metrics not available"})
		return
	}
	hostnames, err := ep.Gateway.metricsStore.GetHostnames()
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	if hostnames == nil {
		hostnames = []string{}
	}
	c.JSON(200, hostnames)
}

func (ep *Endpoints) GET_MetricsMap(c *gin.Context) {
	if ep.Gateway.metricsStore == nil {
		c.JSON(503, gin.H{"error": "metrics not available"})
		return
	}

	from, to, err := parseTimeRange(c)
	if err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	points, err := ep.Gateway.metricsStore.GetMapData(from, to, c.Query("hostname"))
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	if points == nil {
		points = []localmetrics.MapPoint{}
	}
	c.JSON(200, points)
}

func (ep *Endpoints) GET_MetricsTimeSeries(c *gin.Context) {
	if ep.Gateway.metricsStore == nil {
		c.JSON(503, gin.H{"error": "metrics not available"})
		return
	}

	from, to, err := parseTimeRange(c)
	if err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	points, err := ep.Gateway.metricsStore.GetTimeSeries(from, to, c.Query("hostname"), c.DefaultQuery("granularity", "hour"))
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	if points == nil {
		points = []localmetrics.TimePoint{}
	}
	c.JSON(200, points)
}

func (ep *Endpoints) GET_MetricsIPs(c *gin.Context) {
	if ep.Gateway.metricsStore == nil {
		c.JSON(503, gin.H{"error": "metrics not available"})
		return
	}

	from, to, err := parseTimeRange(c)
	if err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	stats, err := ep.Gateway.metricsStore.GetIPStats(from, to, c.Query("hostname"), 100)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	if stats == nil {
		stats = []localmetrics.IPStat{}
	}
	c.JSON(200, stats)
}

func (ep *Endpoints) GET_MetricsPaths(c *gin.Context) {
	if ep.Gateway.metricsStore == nil {
		c.JSON(503, gin.H{"error": "metrics not available"})
		return
	}

	from, to, err := parseTimeRange(c)
	if err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	stats, err := ep.Gateway.metricsStore.GetTopPaths(from, to, c.Query("hostname"), 100)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	if stats == nil {
		stats = []localmetrics.PathStat{}
	}
	c.JSON(200, stats)
}

func parseTimeRange(c *gin.Context) (from, to time.Time, err error) {
	to = time.Now()
	from = to.AddDate(0, 0, -7)

	if fromStr := c.Query("from"); fromStr != "" {
		from, err = time.Parse(time.RFC3339, fromStr)
		if err != nil {
			return
		}
	}
	if toStr := c.Query("to"); toStr != "" {
		to, err = time.Parse(time.RFC3339, toStr)
		if err != nil {
			return
		}
	}
	return
}
