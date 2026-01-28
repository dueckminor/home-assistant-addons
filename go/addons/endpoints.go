package addons

import "github.com/gin-gonic/gin"

type Endpoints interface {
	SetupEndpoints(rg *gin.RouterGroup)
}
