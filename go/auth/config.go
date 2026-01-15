package auth

import (
	"encoding/base64"
	"os"
	"path"

	"github.com/dueckminor/home-assistant-addons/go/utils/crypto"
	"github.com/dueckminor/home-assistant-addons/go/utils/crypto/rand"

	"gopkg.in/yaml.v3"
)

type AuthServerConfig struct {
	AuthKeyBase64 string            `json:"auth_key" yaml:"auth_key"`
	AuthKey       []byte            `json:"-" yaml:"-"`
	EncKeyBase64  string            `json:"enc_key" yaml:"enc_key"`
	EncKey        []byte            `json:"-" yaml:"-"`
	JWTKeyPEM     string            `json:"jwt_key" yaml:"jwt_key"`
	JWTKey        crypto.PrivateKey `json:"-" yaml:"-"`
}

func NewAuthServerConfigFile(filename string) (a *AuthServerConfig, err error) {
	dir := path.Dir(filename)
	err = os.MkdirAll(dir, 0o755)
	if err != nil {
		return nil, err
	}
	a, err = ParseAuthServerConfigFile(filename)
	if err != nil && !os.IsNotExist(err) {
		return nil, err
	}
	if a == nil {
		a = &AuthServerConfig{}
	}

	if a.JWTKeyPEM == "" {
		a.GenerateKeys()
		err = a.Write(filename)
		if err != nil {
			return nil, err
		}
	}

	return a, nil
}

func (a *AuthServerConfig) Write(filename string) error {
	data, err := yaml.Marshal(a)
	if err != nil {
		return err
	}
	err = os.WriteFile(filename, data, 0o644)
	if err != nil {
		return err
	}
	return nil
}

func ParseAuthServerConfigFile(filename string) (a *AuthServerConfig, err error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}
	return ParseAuthServerConfig(data)
}

func ParseAuthServerConfig(data []byte) (a *AuthServerConfig, err error) {
	err = yaml.Unmarshal(data, &a)
	if err != nil {
		return nil, err
	}
	err = a.convertKeys()
	if err != nil {
		return nil, err
	}
	return a, nil
}

func (a *AuthServerConfig) convertKeys() (err error) {
	if a.AuthKeyBase64 != "" {
		a.AuthKey, err = base64.StdEncoding.DecodeString(a.AuthKeyBase64)
		if err != nil {
			return err
		}
	}
	if a.EncKeyBase64 != "" {
		a.EncKey, err = base64.StdEncoding.DecodeString(a.EncKeyBase64)
		if err != nil {
			return err
		}
	}
	if a.JWTKeyPEM != "" {
		a.JWTKey, err = crypto.ParsePrivateKey(a.JWTKeyPEM)
		if err != nil {
			return err
		}
	}
	return nil
}

func (a *AuthServerConfig) GenerateKeys() (err error) {

	if a.AuthKey == nil {
		authKey, err := rand.GetBytes(64)
		if err != nil {
			return err
		}
		a.AuthKey = authKey
		a.AuthKeyBase64 = base64.StdEncoding.EncodeToString(authKey)
	}
	if a.EncKey == nil {
		encKey, err := rand.GetBytes(32)
		if err != nil {
			return err
		}
		a.EncKey = encKey
		a.EncKeyBase64 = base64.StdEncoding.EncodeToString(encKey)
	}
	if a.JWTKey == nil {
		a.JWTKey, err = crypto.CreatePrivateKey()
		if err != nil {
			return err
		}
		a.JWTKeyPEM = a.JWTKey.PEM()
	}
	return nil
}

////////////////////////////////////////////////////////////////////////////////

type AuthClientConfig struct {
	ClientId     string `json:"client_id" yaml:"client_id"`
	ClientSecret string `json:"client_secret" yaml:"client_secret"`
}

type AuthClientConfigManager interface {
	NewAuthClientConfig(clientId string, persistent bool) (c *AuthClientConfig, err error)
	GewAuthClientConfig(clientId string) (c *AuthClientConfig, err error)
}

type authClientConfigManager struct {
	dirname           string
	authClientConfigs map[string]*AuthClientConfig
}

func NewAuthClientConfigManager(dirname string) AuthClientConfigManager {
	return &authClientConfigManager{
		dirname:           dirname,
		authClientConfigs: make(map[string]*AuthClientConfig),
	}
}

func (m *authClientConfigManager) NewAuthClientConfig(clientId string, persistent bool) (c *AuthClientConfig, err error) {
	if c, ok := m.authClientConfigs[clientId]; ok {
		return c, nil
	}

	c = &AuthClientConfig{ClientId: clientId}
	c.ClientSecret, err = rand.GetString(24)
	if err != nil {
		return nil, err
	}

	m.authClientConfigs[clientId] = c

	return c, nil
}

func (m *authClientConfigManager) GewAuthClientConfig(clientId string) (c *AuthClientConfig, err error) {
	if c, ok := m.authClientConfigs[clientId]; ok {
		return c, nil
	}
	return nil, nil
}
