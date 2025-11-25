package network

import (
	"crypto/tls"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"time"

	"github.com/dueckminor/home-assistant-addons/go/auth"
	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
)

type ReverseProxyOptions struct {
	UseTargetHostname bool
	InsecureTLS       bool
	Auth              bool
	AuthClient        *auth.AuthClient
	AuthSecret        string
	SessionStore      sessions.Store
	MetricCallback    MetricCallback
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
		if opt.Auth {
			combinedOptions.Auth = true
		}
		if opt.AuthClient != nil {
			combinedOptions.AuthClient = opt.AuthClient
		}
		if opt.AuthSecret != "" {
			combinedOptions.AuthSecret = opt.AuthSecret
		}
		if opt.SessionStore != nil {
			combinedOptions.SessionStore = opt.SessionStore
		}
		if opt.MetricCallback != nil {
			combinedOptions.MetricCallback = opt.MetricCallback
		}
	}

	if combinedOptions.MetricCallback != nil {
		r.Use(func(c *gin.Context) {
			metric := Metric{
				Timestamp:  time.Now(),
				ClientAddr: c.ClientIP(),
			}
			defer func() {
				metric.Duration = time.Since(metric.Timestamp)
				metric.Hostname = c.Request.Host
				metric.Method = c.Request.Method
				metric.Path = c.Request.URL.Path
				metric.ResponseCode = c.Writer.Status()
				combinedOptions.MetricCallback(metric)
			}()
			c.Next()
		})
	}

	if combinedOptions.AuthClient != nil {
		r.Use(sessions.Sessions("MYPI_ROUTER_SESSION", combinedOptions.SessionStore))
		combinedOptions.AuthClient.RegisterHandler(r)
	}

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

	var tlsConfig *tls.Config

	if options.InsecureTLS {
		tlsConfig = &tls.Config{
			InsecureSkipVerify: true,
		}
		proxy.Transport = &http.Transport{TLSClientConfig: tlsConfig}
	}

	return func(c *gin.Context) {
		if c.IsAborted() {
			return
		}
		req := c.Request
		if options.UseTargetHostname {
			req.Host = hostname
		} else if tlsConfig != nil {
			tlsConfig.ServerName = req.Host
		}
		proxy.ServeHTTP(c.Writer, req)
	}
}
