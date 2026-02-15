package network

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strconv"
	"strings"

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
		r.Use(MetricMiddleware(combinedOptions.MetricCallback))
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

		host := resp.Request.Header.Get("X-Forwarded-Host")
		proto := resp.Request.Header.Get("X-Forwarded-Proto")
		url := proto + "://" + host

		if strings.HasPrefix(host, "bitwarden") || strings.HasPrefix(host, "btwrdn") {
			// Read the original body
			body, err := io.ReadAll(resp.Body)
			if err != nil {
				return nil
			}
			resp.Body.Close()

			fmt.Println(resp.Request.URL.Path + ":")
			fmt.Println("---")
			fmt.Println(string(body))
			fmt.Println("---")

			parsedBody := make(map[string]any)

			err = json.Unmarshal(body, &parsedBody)
			if err != nil || len(body) == 0 {
				resp.Body = io.NopCloser(bytes.NewReader(body))
				resp.ContentLength = int64(len(body))
				resp.Header.Set("Content-Length", strconv.Itoa(len(body)))
				return nil
			}
			if err == nil {

				// recursively replace all strings starting with "http://localhost/path" with target+"/path"
				var replaceURLs func(interface{}) interface{}
				replaceURLs = func(value interface{}) interface{} {
					switch v := value.(type) {
					case string:
						if strings.HasPrefix(v, "http://localhost") {
							return url + v[len("http://localhost"):]
						} else if strings.HasPrefix(v, "https://localhost") {
							return url + v[len("https://localhost"):]
						}
						return v
					case map[string]interface{}:
						for key, val := range v {
							v[key] = replaceURLs(val)
						}
						return v
					case []interface{}:
						for i, val := range v {
							v[i] = replaceURLs(val)
						}
						return v
					default:
						return v
					}
				}

				for key, value := range parsedBody {
					parsedBody[key] = replaceURLs(value)
				}

				// Modify the body here
				modifiedBody, err := json.Marshal(parsedBody)
				if err != nil {
					return err
				}
				fmt.Println(string(modifiedBody))
				fmt.Println("---")

				// Replace the body
				resp.Body = io.NopCloser(bytes.NewReader(modifiedBody))
				resp.ContentLength = int64(len(modifiedBody))
				resp.Header.Set("Content-Length", strconv.Itoa(len(modifiedBody)))
			}
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
