package ginutil

import (
	"embed"
	"net/http/httputil"
	"net/url"
	"strings"

	"github.com/gin-contrib/static"
	"github.com/gin-gonic/gin"
)

func reverseProxy(target string) gin.HandlerFunc {
	url, err := url.Parse(target)
	if err != nil {
		panic(err)
	}
	proxy := httputil.NewSingleHostReverseProxy(url)
	return func(c *gin.Context) {
		proxy.ServeHTTP(c.Writer, c.Request)
	}
}

func ServeFromUri(r *gin.Engine, uri string) {
	if strings.HasPrefix(uri, "http://") {
		// use a reverse proxy - but only for non-API routes
		r.NoRoute(reverseProxy(uri))
	} else {
		// serve static files - but only for non-API routes
		r.NoRoute(static.ServeRoot("/", uri))
	}
}

func ServeEmbedFS(r *gin.Engine, embedFS embed.FS, targetPath string) {
	fs, _ := static.EmbedFolder(embedFS, targetPath)
	r.NoRoute(static.Serve("/", fs))
}
