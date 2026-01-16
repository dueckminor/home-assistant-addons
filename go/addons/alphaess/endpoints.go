package alphaess

import "github.com/gin-gonic/gin"

type Endpoints struct {
	addon *Addon
}

func NewEndpoints(a *Addon) *Endpoints {
	return &Endpoints{
		addon: a,
	}
}

func (e *Endpoints) SetupEndpoints(rg *gin.RouterGroup) {
	rg.GET("/status", e.getStatus)
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
