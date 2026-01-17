package alphaess

import (
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
	rg.GET("/gaps", e.getGaps)
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
