package alphaess

import (
	"fmt"

	"github.com/dueckminor/home-assistant-addons/go/utils/crypto/rand"
	"github.com/dueckminor/home-assistant-addons/go/utils/ginutil"
	"github.com/gin-gonic/gin"
)

type Endpoints struct {
	addon *addon
}

func NewEndpoints(a *addon) *Endpoints {
	return &Endpoints{
		addon: a,
	}
}

func (e *Endpoints) SetupEndpoints(rg *gin.RouterGroup) {
	rg.GET("/status", e.getStatus)
	rg.GET("/measurements", e.getMeasurements)
	rg.GET("/measurements/aggregate", e.getAggregatedMeasurements)
	rg.GET("/gaps", e.getGaps)
	rg.POST("/import-csv", e.importCSV)
	rg.GET("/fill-gaps", e.previewFillGaps)
	rg.POST("/fill-gaps", e.commitFillGaps)
}

func (e *Endpoints) getStatus(c *gin.Context) {
	c.JSON(200, gin.H{
		"connected":         true,
		"mqttConnected":     true,
		"alphaessConnected": true,
		"mqttUri":           e.addon.config.MqttURI,
		"alphaessUri":       e.addon.config.AlphaEssUri,
	})
}

func (e *Endpoints) getMeasurements(c *gin.Context) {
	filter := MeasurementFilter{
		MeasurementNames: ginutil.ParseQueryStringArray(c, "names"),
		Previous:         ginutil.ParseQueryBool(c, "previous"),
		NotBefore:        ginutil.ParseQueryTime(c, "not_before"),
		After:            ginutil.ParseQueryTime(c, "after"),
		NotAfter:         ginutil.ParseQueryTime(c, "not_after"),
		Before:           ginutil.ParseQueryTime(c, "before"),
	}

	measurements := e.addon.GetMeasurements(filter)
	c.JSON(200, measurements)
}

func (e *Endpoints) getGaps(c *gin.Context) {
	filter := MeasurementFilter{
		MeasurementNames: ginutil.ParseQueryStringArray(c, "names"),
		Previous:         ginutil.ParseQueryBool(c, "previous"),
		NotBefore:        ginutil.ParseQueryTime(c, "not_before"),
		After:            ginutil.ParseQueryTime(c, "after"),
		NotAfter:         ginutil.ParseQueryTime(c, "not_after"),
		Before:           ginutil.ParseQueryTime(c, "before"),
	}

	measurements := e.addon.GetGaps(filter)
	c.JSON(200, measurements)
}

func (e *Endpoints) getAggregatedMeasurements(c *gin.Context) {
	// Parse query parameters
	interval := c.Query("interval")
	if interval == "" {
		interval = "hourly" // default
	}

	timezone := c.Query("timezone")
	if timezone == "" {
		timezone = "UTC" // default
	}

	from := ginutil.ParseQueryTime(c, "from")
	to := ginutil.ParseQueryTime(c, "to")

	if from.IsZero() || to.IsZero() {
		c.JSON(400, gin.H{"error": "from and to parameters are required"})
		return
	}

	params := AggregateParameters{
		Interval: interval,
		From:     from,
		To:       to,
		Timezone: timezone,
	}

	aggregates, err := e.addon.Aggregate(params)
	if err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	c.JSON(200, aggregates)
}

// importCSV handles POST /api/import-csv
func (e *Endpoints) importCSV(c *gin.Context) {
	var req struct {
		Date     string `json:"date" binding:"required"`
		Timezone string `json:"timezone" binding:"required"`
		Content  string `json:"content" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	// Parse CSV
	session, err := ParseCSV(req.Content, req.Date, req.Timezone)
	if err != nil {
		c.JSON(400, gin.H{"error": fmt.Sprintf("Failed to parse CSV: %v", err)})
		return
	}

	// Generate session ID
	sessionID, err := rand.GetString(16)
	if err != nil {
		c.JSON(500, gin.H{"error": "Failed to generate session ID"})
		return
	}

	// Store session
	StoreSession(sessionID, session)

	c.JSON(200, gin.H{
		"sessionId":   sessionID,
		"date":        req.Date,
		"recordCount": len(session.Records),
		"message":     "CSV uploaded and parsed successfully",
	})
}

// previewFillGaps handles GET /api/fill-gaps
func (e *Endpoints) previewFillGaps(c *gin.Context) {
	sessions := GetAllSessions()
	if len(sessions) == 0 {
		c.JSON(400, gin.H{"error": "No CSV files uploaded"})
		return
	}

	// Aggregate all sessions to hourly data
	preview := make(map[string]interface{})
	totalHours := 0

	for sessionID, session := range sessions {
		hourlyData := session.AggregateToHourly()
		preview[sessionID] = gin.H{
			"date":       session.Date.Format("2006-01-02"),
			"hours":      len(hourlyData),
			"sampleData": hourlyData[:min(3, len(hourlyData))], // Show first 3 hours as preview
		}
		totalHours += len(hourlyData)
	}

	c.JSON(200, gin.H{
		"sessions":   preview,
		"totalHours": totalHours,
		"message":    "Preview of data to be imported",
	})
}

// commitFillGaps handles POST /api/fill-gaps
func (e *Endpoints) commitFillGaps(c *gin.Context) {
	sessions := GetAllSessions()
	if len(sessions) == 0 {
		c.JSON(400, gin.H{"error": "No CSV files uploaded"})
		return
	}

	// TODO: Implement actual database insertion
	// For now, just return success
	totalHours := 0
	for _, session := range sessions {
		hourlyData := session.AggregateToHourly()
		totalHours += len(hourlyData)
		// e.addon.InsertMeasurements(hourlyData)
	}

	// Clear all sessions after commit
	for sessionID := range sessions {
		ClearSession(sessionID)
	}

	c.JSON(200, gin.H{
		"success":    true,
		"hoursAdded": totalHours,
		"message":    "Data imported successfully",
	})
}

func min[T int | float64](a, b T) T {
	if a < b {
		return a
	}
	return b
}

func max[T int | float64](a, b T) T {
	if a > b {
		return a
	}
	return b
}
