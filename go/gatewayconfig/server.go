package gatewayconfig

import (
	"embed"
	"net"
	"time"

	"github.com/dueckminor/home-assistant-addons/go/ginutil"
	"github.com/gin-gonic/gin"
)

//go:embed dist/*
var distFS embed.FS

func NewGatewayConfigServer(r *gin.Engine, distDir string) (g *GatewayConfigServer, err error) {
	g = new(GatewayConfigServer)

	if distDir != "" {
		ginutil.ServeFromUri(r, distDir)
	} else {
		ginutil.ServeEmbedFS(r, distFS, "dist")
	}

	setupDNSAPI(r)

	return g, nil
}

type GatewayConfigServer struct {
}

func setupDNSAPI(r *gin.Engine) {
	api := r.Group("/api/dns")

	api.GET("/ipv4/:hostname", func(c *gin.Context) {
		hostname := c.Param("hostname")
		ips, err := net.LookupIP(hostname)
		if err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}

		for _, ip := range ips {
			if ip.To4() != nil { // IPv4
				c.JSON(200, gin.H{"ip": ip.String(), "timestamp": time.Now()})
				return
			}
		}
		c.JSON(404, gin.H{"error": "No IPv4 found"})
	})

	api.GET("/ipv6/:hostname", func(c *gin.Context) {
		hostname := c.Param("hostname")
		ips, err := net.LookupIP(hostname)
		if err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}

		for _, ip := range ips {
			if ip.To4() == nil { // IPv6
				c.JSON(200, gin.H{"ip": ip.String(), "timestamp": time.Now()})
				return
			}
		}
		c.JSON(404, gin.H{"error": "No IPv6 found"})
	})
}
