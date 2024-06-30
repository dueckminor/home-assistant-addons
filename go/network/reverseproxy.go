package network

import (
	"crypto/tls"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"

	"github.com/dueckminor/home-assistant-addons/go/auth"
	"github.com/gin-gonic/gin"
)

type ReverseProxyOptions struct {
	UseTargetHostname bool
	InsecureTLS       bool
	AuthClient        auth.AuthClient
}

func ParseReverseProxyOptions(options string) (result ReverseProxyOptions) {
	for _, option := range strings.Split(options, ",") {
		if option == "insecure" {
			result.InsecureTLS = true
		}
		if option == "use-target-hostname" {
			result.UseTargetHostname = true
		}
	}
	return result
}

func NewHostImplReverseProxy(uri string, options ...ReverseProxyOptions) ServeCtx {
	r := gin.Default()

	combinedOptions := ReverseProxyOptions{}

	for _, opt := range options {
		if opt.UseTargetHostname {
			combinedOptions.UseTargetHostname = true
		}
		if opt.InsecureTLS {
			combinedOptions.InsecureTLS = true
		}
	}

	// if ac != nil {
	// 	h.r.Use(sessions.Sessions("MYPI_ROUTER_SESSION", store))
	// 	ac.RegisterHandler(h.r)
	// }

	r.Use(SingleHostReverseProxy(uri, combinedOptions))

	return NewGinHandler(r)
}

func SingleHostReverseProxy(target string, options ReverseProxyOptions) gin.HandlerFunc {
	url, _ := url.Parse(target)
	hostname := url.Hostname()
	proxy := &httputil.ReverseProxy{
		Rewrite: func(r *httputil.ProxyRequest) {
			r.SetURL(url)
			r.SetXForwarded()
			r.Out.Host = r.In.Host
		},
	}

	proxy.ModifyResponse = func(resp *http.Response) error {
		location := resp.Header.Get("Location")
		if strings.HasPrefix(location, target) {
			newLocation := location[len(target):]
			resp.Header.Set("Location", newLocation)
		}
		return nil
	}
	if options.InsecureTLS {
		proxy.Transport = &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		}
	}

	return func(c *gin.Context) {
		if c.IsAborted() {
			return
		}
		req := c.Request
		if options.UseTargetHostname {
			req.Host = hostname
		}
		proxy.ServeHTTP(c.Writer, req)
	}
}
