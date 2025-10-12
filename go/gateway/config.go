package gateway

import (
	"fmt"
	"os"

	"github.com/dueckminor/home-assistant-addons/go/pki"
	"github.com/goccy/go-yaml"
	"github.com/google/uuid"
)

type ConfigExternalIp struct {
	Source  string `yaml:"source" json:"source"`
	Options string `yaml:"options" json:"options"`
}

type ConfigRouteOptions struct {
	Insecure          bool   `yaml:"insecure,omitempty" json:"insecure,omitempty"`
	UseTargetHostname bool   `yaml:"use_target_hostname,omitempty" json:"use_target_hostname,omitempty"`
	Auth              bool   `yaml:"auth,omitempty" json:"auth,omitempty"`
	AuthSecret        string `yaml:"auth_secret,omitempty" json:"auth_secret,omitempty"`
}

type ConfigRoute struct {
	Guid     string             `yaml:"guid" json:"guid"`
	Hostname string             `yaml:"hostname" json:"hostname"`
	Target   string             `yaml:"target" json:"target"`
	Options  ConfigRouteOptions `yaml:"options" json:"options"`
	domain   *ConfigDomain
}

func (configRoute *ConfigRoute) GetHostname() string {
	return configRoute.Hostname + "." + configRoute.domain.Name
}

type ConfigRedirect struct {
	Target    string `yaml:"target" json:"target"`
	HttpPort  int    `yaml:"http_port" json:"http_port"`
	HttpsPort int    `yaml:"https_port" json:"https_port"`
	DnsPort   int    `yaml:"dns_port" json:"dns_port"`
}

func (configRedirect *ConfigRedirect) GetHTTPSTarget() string {
	return fmt.Sprintf("tcp://%s:%d", configRedirect.Target, configRedirect.HttpsPort)
}

func (configRedirect *ConfigRedirect) GetDNSTarget() string {
	return fmt.Sprintf("udp://%s:%d", configRedirect.Target, configRedirect.DnsPort)
}

type ConfigDomain struct {
	Guid     string          `yaml:"guid" json:"guid"`
	Name     string          `yaml:"name" json:"name"`
	Routes   []*ConfigRoute  `yaml:"routes,omitempty" json:"routes,omitempty"`
	Redirect *ConfigRedirect `yaml:"redirect,omitempty" json:"redirect,omitempty"`

	serverCertificate pki.ServerCertificate
}

func (configDomain *ConfigDomain) AddRoute(route *ConfigRoute) {
	route.domain = configDomain
	configDomain.Routes = append(configDomain.Routes, route)
}

func (configDomain *ConfigDomain) GetRoute(guid string) *ConfigRoute {
	for _, route := range configDomain.Routes {
		if route.Guid == guid {
			return route
		}
	}
	return nil
}

func (configDomain *ConfigDomain) GetRouteByHostname(hostname string) *ConfigRoute {
	for _, route := range configDomain.Routes {
		if route.Hostname == hostname {
			return route
		}
	}
	return nil
}

func (configDomain *ConfigDomain) DeleteRoute(guid string) *ConfigRoute {
	for i, route := range configDomain.Routes {
		if route.Guid == guid {
			configDomain.Routes = append(configDomain.Routes[:i], configDomain.Routes[i+1:]...)
			return route
		}
	}
	return nil
}

type ConfigDns struct {
	ExternalIpv4 ConfigExternalIp `yaml:"external_ipv4" json:"external_ipv4"`
	ExternalIpv6 ConfigExternalIp `yaml:"external_ipv6" json:"external_ipv6"`
}

type ConfigMail struct {
	Enabled   bool   `yaml:"enabled" json:"enabled"`
	Email     string `yaml:"email" json:"email"`
	Password  string `yaml:"password" json:"password"`
	SmtpHost  string `yaml:"smtp_host" json:"smtp_host"`
	SmtpPort  int    `yaml:"smtp_port" json:"smtp_port"`
	UseTLS    bool   `yaml:"use_tls" json:"use_tls"`
	FromEmail string `yaml:"from_email" json:"from_email"`
	FromName  string `yaml:"from_name" json:"from_name"`
}

type Config struct {
	file    string
	Domains []*ConfigDomain `yaml:"domains" json:"domains"`
	Dns     ConfigDns       `yaml:"dns" json:"dns"`
	Mail    ConfigMail      `yaml:"mail" json:"mail"`
}

func (config *Config) GetDomain(guid string) *ConfigDomain {
	for _, domain := range config.Domains {
		if domain.Guid == guid {
			return domain
		}
	}
	return nil
}

func (config *Config) DeleteDomain(guid string) *ConfigDomain {
	for i, domain := range config.Domains {
		if domain.Guid == guid {
			config.Domains = append(config.Domains[:i], config.Domains[i+1:]...)
			return domain
		}
	}
	return nil
}

func (config *Config) GetDomainByName(name string) *ConfigDomain {
	for _, domain := range config.Domains {
		if domain.Name == name {
			return domain
		}
	}
	return nil
}

func (config *Config) GetAuthRoute() *ConfigRoute {
	for _, domain := range config.Domains {
		for _, route := range domain.Routes {
			if route.Target == "@auth" {
				return route
			}
		}
	}
	return nil
}

func loadConfig(file string) (*Config, error) {
	configYaml, err := os.ReadFile(file)
	if err != nil {
		if !os.IsNotExist(err) {
			return nil, err
		}
	}

	var config Config

	err = yaml.Unmarshal(configYaml, &config)
	if err != nil {
		return nil, err
	}

	mustSave := false

	for _, domain := range config.Domains {
		if domain.Guid == "" {
			domain.Guid = uuid.New().String()
			mustSave = true
		}
		for _, route := range domain.Routes {
			route.domain = domain
			if route.Guid == "" {
				route.Guid = uuid.New().String()
				mustSave = true
			}
		}
	}

	config.file = file

	if mustSave {
		config.save()
	}

	return &config, nil
}

func (config *Config) save() error {
	configYaml, err := yaml.Marshal(config)
	if err != nil {
		return err
	}

	err = os.WriteFile(config.file+".new", configYaml, 0644)
	if err != nil {
		return err
	}

	err = os.Rename(config.file+".new", config.file)
	if err != nil {
		return err
	}

	return nil
}
