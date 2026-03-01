package auth

import (
	"crypto"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/dueckminor/home-assistant-addons/go/utils/ginutil"
	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt"
)

type AuthClient struct {
	AuthURI      string
	ClientID     string
	ClientSecret string
	ServerKey    crypto.PublicKey
	Secret       string
}

func (ac *AuthClient) RegisterHandler(e *gin.Engine) {
	e.GET("/login/callback", ac.handleLoginCallback)
	e.Use(ac.handleAuth)
}

func (ac *AuthClient) verifySession(c *gin.Context) bool {
	hostname := ginutil.GetHostname(c)

	session := sessions.Default(c)

	accessToken := session.Get("access_token")
	if nil != accessToken {
		if hostname != session.Get("hostname") {
			c.AbortWithStatus(http.StatusInternalServerError)
			return false
		}
		return true
	}
	return false
}

func (ac *AuthClient) handleAuth(c *gin.Context) {
	if c.IsAborted() {
		return
	}
	sessionVerified := ac.verifySession(c)

	if c.Request.URL.Path == "/flv" && c.Request.Method == "GET" {
		return
	}

	if ac.Secret != "" {
		sessionVerified = ac.handleSecret(c, sessionVerified)
	}

	if sessionVerified {
		return
	}

	scheme := ginutil.GetScheme(c)
	hostname := ginutil.GetHostname(c)

	authRequest, err := NewRequest()
	if err != nil {
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	authRequest.Path = c.Request.URL.Path

	callbackURI := &url.URL{
		Scheme: scheme,
		Host:   hostname,
		Path:   "/login/callback",
	}
	values := url.Values{}
	values.Add("id", authRequest.Id)
	callbackURI.RawQuery = values.Encode()

	authRequest.RedirectURI = callbackURI.String()

	redirectToAuthURI, _ := url.Parse(ac.AuthURI)
	values = redirectToAuthURI.Query()
	values.Add("redirect_uri", authRequest.RedirectURI)
	values.Add("response_type", "code")
	values.Add("client_id", ac.ClientID)
	redirectToAuthURI.Path = "/oauth/authorize"
	redirectToAuthURI.RawQuery = values.Encode()

	c.Header("Location", redirectToAuthURI.String())
	c.AbortWithStatus(http.StatusFound)
}

func (ac *AuthClient) handleSecret(c *gin.Context, sessionVerified bool) bool {
	if !sessionVerified {
		secret := c.Request.URL.Query().Get("mypi-secret")
		if secret == "" {
			secret = c.Request.URL.Query().Get("secret")
		}
		if secret == ac.Secret {
			session := sessions.Default(c)
			session.Set("access_token", "anonymous")
			session.Set("hostname", ginutil.GetHostname(c))
			err := session.Save()
			if err != nil {
				c.AbortWithStatus(http.StatusInternalServerError)
				return false
			}
			return true
		}
	}

	path := c.Request.URL.Path
	pattern := "/secret/" + ac.Secret
	if path == pattern || strings.HasPrefix(path, pattern+"/") {
		if !sessionVerified {
			session := sessions.Default(c)
			session.Set("access_token", "anonymous")
			session.Set("hostname", ginutil.GetHostname(c))
			err := session.Save()
			if err != nil {
				c.AbortWithStatus(http.StatusInternalServerError)
				return false
			}
		}
		redirect := "/"
		if path != pattern {
			redirect = path[len(pattern):]
		}

		if c.Request.Method == "GET" {
			c.Header("Location", redirect)
			c.AbortWithStatus(http.StatusFound)
		} else {
			c.Request.URL.Path = redirect
		}
		return true
	}
	return sessionVerified
}

func (ac *AuthClient) handleLoginCallback(c *gin.Context) {
	fmt.Println("login callback 2")
	session := sessions.Default(c)
	err := c.Request.ParseForm()
	if err != nil {
		c.AbortWithStatus(http.StatusBadRequest)
		return
	}
	code := c.Request.Form.Get("code")
	id := c.Request.Form.Get("id")

	authRequest := GetRequest(id)
	if nil == authRequest {
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	path := authRequest.Path

	v := url.Values{}
	v.Set("grant_type", "authorization_code")
	v.Set("code", code)
	v.Set("response_type", "token")
	v.Set("redirect_uri", authRequest.RedirectURI)
	v.Set("client_id", ac.ClientID)

	//pass the values to the request's body

	authURIOauthToken, _ := url.Parse(ac.AuthURI)
	authURIOauthToken.Path = "oauth/token"

	req, err := http.NewRequest("POST", authURIOauthToken.String(), strings.NewReader(v.Encode()))
	if err != nil {
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}
	req.SetBasicAuth(ac.ClientID, ac.ClientSecret)
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Add("Accept", "application/json")
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	bodyText, err := io.ReadAll(resp.Body)
	if err != nil {
		c.AbortWithStatus(http.StatusBadRequest)
		return
	}

	var data map[string]any
	err = json.Unmarshal(bodyText, &data)
	if err != nil {
		c.AbortWithStatus(http.StatusBadRequest)
		return
	}

	jwtToken, ok := data["access_token"].(string)
	if !ok {
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	token, err := jwt.Parse(jwtToken, func(token *jwt.Token) (any, error) {
		if _, ok := token.Method.(*jwt.SigningMethodRSA); ok {
			return ac.ServerKey, nil
		}
		return nil, fmt.Errorf("Unexpected Signing Method")
	})
	if err != nil {
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}
	fmt.Println(token)

	session.Set("access_token", data["access_token"])
	session.Set("hostname", ginutil.GetHostname(c))
	err = session.Save()
	if err != nil {
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	c.Header("Location", path)
	c.AbortWithStatus(http.StatusFound)
}
