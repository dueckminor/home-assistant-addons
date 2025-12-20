package auth

import (
	"crypto"
	"embed"
	"encoding/base64"
	"net"
	"net/http"
	"net/url"
	"path"
	"strings"

	"github.com/dueckminor/home-assistant-addons/go/crypto/rand"
	"github.com/dueckminor/home-assistant-addons/go/ginutil"
	"github.com/dueckminor/home-assistant-addons/go/smtp"

	"github.com/gin-contrib/cors"
	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt"
)

//go:embed dist/*
var distFS embed.FS

func NewAuthServer(r *gin.Engine, distDir string, dataDir string) (a *AuthServer, err error) {
	a = new(AuthServer)
	a.dataDir = dataDir
	a.clients = NewAuthClientConfigManager(path.Join(dataDir, "clients"))

	a.config, err = NewAuthServerConfigFile(path.Join(dataDir, "server.yml"))
	if err != nil {
		return nil, err
	}

	a.users, err = NewUsers(dataDir)
	if err != nil {
		return nil, err
	}

	if distDir != "" {
		ginutil.ServeFromUri(r, distDir)
	} else {
		ginutil.ServeEmbedFS(r, distFS, "dist")
	}

	a.Register(r)
	return a, nil
}

type AuthServer struct {
	dataDir      string
	config       *AuthServerConfig
	clients      AuthClientConfigManager
	sessionStore sessions.Store
	users        Users
	// for the password reset
	hostname   string
	domain     string
	smtpClient *smtp.Client
}

func (a *AuthServer) Register(r *gin.Engine) {

	store := a.GetSessionStore()

	// Configure CORS to allow the 'next' header and credentials
	config := cors.DefaultConfig()
	config.AllowAllOrigins = true
	config.AllowCredentials = true
	config.AllowHeaders = append(config.AllowHeaders, "next")
	r.Use(cors.New(config))

	rg := r.Group("")
	rg.Use(sessions.Sessions("MYPI_AUTH_SESSION", store))
	rg.POST("/login", a.login)
	rg.POST("/logout", a.handleLogout)
	rg.GET("/status", a.handleStatus)
	rg.GET("/oauth/authorize", a.handleOauthAuthorize)
	rg.POST("/oauth/token", a.handleOauthToken)
	rg.POST("/send_reset_password_mail", a.sendResetPasswordMail)
	rg.POST("/reset_password", a.resetPassword)
}

func (a *AuthServer) Users() Users {
	return a.users
}

func (a *AuthServer) GetPublicKey() (p crypto.PublicKey) {
	return a.config.JWTKey.Public()
}

func (a *AuthServer) GetAuthClientConfig(clientId string) (c *AuthClientConfig, err error) {
	return a.clients.NewAuthClientConfig(clientId, false)
}

func (a *AuthServer) GetSessionStore() sessions.Store {
	if a.sessionStore == nil {
		a.sessionStore = cookie.NewStore(a.config.AuthKey, a.config.EncKey)
	}
	return a.sessionStore
}

func (a *AuthServer) EnableSMTP(hostname string, domain string, smtpClient *smtp.Client) {
	a.hostname = hostname
	a.domain = domain
	a.smtpClient = smtpClient
}

func (a *AuthServer) login(c *gin.Context) {
	var params struct {
		Username string
		Password string
	}
	err := c.BindJSON(&params)
	if err != nil {
		c.AbortWithStatus(http.StatusBadRequest)
		return
	}

	if !a.users.CheckPassword(params.Username, params.Password) {
		c.AbortWithStatus(http.StatusUnauthorized)
		return
	}

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

func (a *AuthServer) handleOauthAuthorize(c *gin.Context) {
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

func (a *AuthServer) basicAuth(c *gin.Context) string {
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

	clientConfig, err := a.clients.GewAuthClientConfig(clientID)
	if err != nil {
		return ""
	}
	if clientConfig.ClientSecret != pair[1] {
		return ""
	}

	return clientID
}

func (a *AuthServer) handleOauthToken(c *gin.Context) {
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

	username := a.basicAuth(c)
	if username != clientID {
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	authRequest := GetRequest(code)
	if authRequest.RedirectURI != redirectURI {
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	token := jwt.NewWithClaims(jwt.SigningMethodRS256, ClaimsWithScope{})
	jwt, _ := token.SignedString(a.config.JWTKey.RSA())

	response := OauthTokenResponse{
		AccessToken: jwt,
	}
	c.AbortWithStatusJSON(http.StatusOK, response)
}

func (a *AuthServer) handleLogout(c *gin.Context) {
	session := sessions.Default(c)
	session.Clear()
	err := session.Save()
	if err != nil {
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}
	c.AbortWithStatus(http.StatusAccepted)
}

func (a *AuthServer) sendResetPasswordMail(c *gin.Context) {
	payload := struct {
		Mail string
	}{}

	c.BindJSON(&payload)

	if payload.Mail == "" {
		c.AbortWithStatus(http.StatusBadRequest)
		return
	}

	if a.smtpClient == nil {
		c.AbortWithStatus(http.StatusForbidden)
		return
	}

	go func() {

		user, err := a.Users().StartPasswordReset(payload.Mail)
		if err != nil {
			return
		}

		resetURL := "https://" + a.hostname + "?password_reset=" + user.ResetToken
		a.smtpClient.SendPasswordResetEmail(user.Mail, user.ResetToken, resetURL)
	}()

	c.AbortWithStatus(http.StatusAccepted)
}

func (a *AuthServer) resetPassword(c *gin.Context) {
	payload := struct {
		Token    string
		Password string
	}{}

	c.BindJSON(&payload)

	if payload.Token == "" || payload.Password == "" {
		c.AbortWithStatus(http.StatusBadRequest)
		return
	}

	err := a.Users().PasswordReset(payload.Token, payload.Password)
	if err != nil {
		c.AbortWithStatus(http.StatusForbidden)
		return
	}

	c.AbortWithStatus(http.StatusAccepted)
}

type status struct {
	Username string `json:"username"`
}

func (a *AuthServer) handleStatus(c *gin.Context) {
	session := sessions.Default(c)

	username, _ := session.Get("username").(string)

	c.AbortWithStatusJSON(http.StatusOK, status{
		Username: username,
	})
}
