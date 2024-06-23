package auth

import (
	"encoding/base64"
	"net"
	"net/http"
	"net/url"
	"strings"

	"github.com/dueckminor/home-assistant-addons/go/crypto/rand"

	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt"
)

func login(c *gin.Context) {
	var params struct {
		Username string
		Password string
	}
	err := c.BindJSON(&params)
	if err != nil {
		c.AbortWithStatus(http.StatusBadRequest)
		return
	}

	// if !users.CheckPassword(params.Username, params.Password) {
	// 	c.AbortWithStatus(http.StatusUnauthorized)
	// 	return
	// }

	session := sessions.Default(c)

	secret, _ := session.Get("secret").(string)

	if len(secret) == 0 {
		domain := ""
		host, _, _ := net.SplitHostPort(c.Request.Host)
		addr := net.ParseIP(host)
		if addr != nil {
			origin := c.Request.Header["Origin"]
			if len(origin) == 1 {
				uri, _ := url.Parse(origin[0])
				host = uri.Hostname()
				addr = net.ParseIP(host)
			}
		}

		if addr == nil {
			hostParts := strings.Split(host, ".")
			if len(hostParts) > 1 {
				domain = strings.Join(hostParts[1:], ".")
			}
		}

		secret, _ = rand.GetString(48)
		session.Set("secret", secret)
		session.Set("domain", domain)
		session.Set("username", params.Username)
		err = session.Save()
		if err != nil {
			c.AbortWithStatus(http.StatusInternalServerError)
		}
	}
	c.Data(http.StatusOK, "text/plain", []byte("OK"))
}

type ClaimsWithScope struct {
	Scopes []string `json:"scopes,omitempty"`
	jwt.StandardClaims
}

func handleOauthAuthorize(c *gin.Context) {
	session := sessions.Default(c)
	secret, _ := session.Get("secret").(string)

	if len(secret) > 0 {
		authRequest, err := NewRequest()
		if err != nil {
			c.AbortWithStatus(http.StatusInternalServerError)
			return
		}

		values := c.Request.URL.Query()
		authRequest.RedirectURI = values.Get("redirect_uri")
		redirectURI, err := url.Parse(authRequest.RedirectURI)
		if err != nil {
			c.AbortWithStatus(http.StatusInternalServerError)
			return
		}
		values = redirectURI.Query()
		values.Add("code", authRequest.Id)
		redirectURI.RawQuery = values.Encode()
		c.Header("Location", redirectURI.String())
		c.AbortWithStatus(http.StatusFound)
	}

	c.Request.URL.Path = "/"
	c.Header("Location", c.Request.URL.String())
	c.AbortWithStatus(http.StatusFound)
}

type OauthTokenResponse struct {
	AccessToken string `json:"access_token"`
}

func basicAuth(c *gin.Context) string {
	s := strings.SplitN(c.Request.Header.Get("Authorization"), " ", 2)
	if len(s) != 2 {
		return ""
	}
	b, err := base64.StdEncoding.DecodeString(s[1])
	if err != nil {
		return ""
	}
	pair := strings.SplitN(string(b), ":", 2)
	if len(pair) != 2 {
		return ""
	}

	clientID := pair[0]
	if len(clientID) == 0 {
		return ""
	}
	if strings.Contains(clientID, "..") || strings.ContainsAny(clientID, "/\\") {
		return ""
	}

	// clientConfig, err := config.ReadConfigFile(dirMypiAuthClients, clientID+".yml")
	// if err != nil {
	// 	return ""
	// }
	// clientSecret := clientConfig.GetString("client_secret")
	// if clientSecret != pair[1] {
	// 	return ""
	// }

	return clientID
}

func handleOauthToken(c *gin.Context) {
	err := c.Request.ParseForm()
	if err != nil {
		c.AbortWithStatus(http.StatusBadRequest)
		return
	}
	code := c.Request.Form.Get("code")
	grantType := c.Request.Form.Get("grant_type")
	responseType := c.Request.Form.Get("response_type")
	redirectURI := c.Request.Form.Get("redirect_uri")
	clientID := c.Request.Form.Get("client_id")

	if grantType != "authorization_code" {
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}
	if responseType != "token" {
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	username := basicAuth(c)
	if username != clientID {
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	authRequest := GetRequest(code)
	if authRequest.RedirectURI != redirectURI {
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	// token := jwt.NewWithClaims(jwt.SigningMethodRS256, ClaimsWithScope{})
	// key, _ := config.ReadRSAPrivateKey("etc/mypi-auth/server/server_priv.pem")
	// jwt, _ := token.SignedString(key)

	response := OauthTokenResponse{
		//AccessToken: jwt,
	}
	c.AbortWithStatusJSON(http.StatusOK, response)
}

func handleLogout(c *gin.Context) {
	session := sessions.Default(c)
	session.Clear()
	err := session.Save()
	if err != nil {
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}
	c.AbortWithStatus(http.StatusAccepted)
}

type status struct {
	Username string `json:"username"`
}

func handleStatus(c *gin.Context) {
	session := sessions.Default(c)

	username, _ := session.Get("username").(string)

	c.AbortWithStatusJSON(http.StatusOK, status{
		Username: username,
	})
}
