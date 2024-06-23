package network

import (
	"context"
	"crypto/tls"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"

	"github.com/gin-gonic/gin"
)

type ReverseProxyOptions struct {
	UseTargetHostname bool
	InsecureTLS       bool
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

type HostImplReverseProxy struct {
	listener Listener
	r        *gin.Engine
	options  ReverseProxyOptions
}

func (h *HostImplReverseProxy) Serve(conn net.Conn) {
	// no need to do this here: defer conn.Close()
	// the connection will be closed by the gin.Engine
	h.listener <- conn
}
func (h *HostImplReverseProxy) ServeCtx(ctx context.Context, conn net.Conn) {
	// no need to do this here: defer conn.Close()
	// the connection will be closed by the gin.Engine
	h.listener <- conn
}

func NewHostImplReverseProxy(uri string, options ...ReverseProxyOptions) *HostImplReverseProxy {
	h := new(HostImplReverseProxy)
	h.r = gin.Default()

	for _, opt := range options {
		if opt.UseTargetHostname {
			h.options.UseTargetHostname = true
		}
		if opt.InsecureTLS {
			h.options.InsecureTLS = true
		}
	}

	// if ac != nil {
	// 	h.r.Use(sessions.Sessions("MYPI_ROUTER_SESSION", store))
	// 	ac.RegisterHandler(h.r)
	// }

	h.listener = MakeListener()
	go h.r.RunListener(h.listener) // nolint:errcheck
	h.r.Use(SingleHostReverseProxy(uri, h.options))
	return h
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
