package gatewayconfig

import (
	"embed"

	"github.com/gin-contrib/static"
	"github.com/gin-gonic/gin"
)

//go:embed dist/*
var distFS embed.FS

func NewGatewayConfigServer(r *gin.Engine, distDir string) (g *GatewayConfigServer, err error) {
	g = new(GatewayConfigServer)

	fs, _ := static.EmbedFolder(distFS, "dist")
	r.Use(static.Serve("/", fs))

	r.GET("/", func(c *gin.Context) {
		data, _ := distFS.ReadFile("dist/index.html")
		c.Data(200, "text/html", data)
	})
	r.GET("/index.html", func(c *gin.Context) {
		data, _ := distFS.ReadFile("dist/index.html")
		c.Data(200, "text/html", data)
	})

	return g, nil
}

type GatewayConfigServer struct {
}
