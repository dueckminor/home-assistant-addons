package gateway

import (
	"os"

	"github.com/goccy/go-yaml"
)

type ConfigExternalIp struct {
	Source  string `yaml:"source" json:"source"`
	Options string `yaml:"options" json:"options"`
}

type ConfigRoute struct {
	Guid     string `yaml:"guid" json:"guid"`
	Hostname string `yaml:"hostname" json:"hostname"`
	Target   string `yaml:"target" json:"target"`
	Mode     string `yaml:"mode" json:"mode"`
}

type ConfigDev struct {
	Domain      string `yaml:"domain" json:"domain"`
	HttpTarget  string `yaml:"http_target" json:"http_target"`
	HttpsTarget string `yaml:"https_target" json:"https_target"`
	DnsTarget   string `yaml:"dns_target" json:"dns_target"`
}

type ConfigDomain struct {
	Guid   string        `yaml:"guid" json:"guid"`
	Name   string        `yaml:"name" json:"name"`
	Routes []ConfigRoute `yaml:"routes" json:"routes"`
}

type ConfigDns struct {
	ExternalIpv4 ConfigExternalIp `yaml:"external_ipv4" json:"external_ipv4"`
	ExternalIpv6 ConfigExternalIp `yaml:"external_ipv6" json:"external_ipv6"`
}

type Config struct {
	file    string
	Domains []ConfigDomain `yaml:"domains" json:"domains"`
	Dns     ConfigDns      `yaml:"dns" json:"dns"`
	Dev     ConfigDev      `yaml:"dev" json:"dev"`
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

	config.file = file
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
