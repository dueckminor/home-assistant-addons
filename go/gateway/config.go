package gateway

import (
	"os"

	"github.com/goccy/go-yaml"
)

type ConfigExternalIp struct {
	Source  string `yaml:"source"`
	Options string `yaml:"options"`
}

type ConfigServer struct {
	Hostname string `yaml:"hostname"`
	Target   string `yaml:"target"`
	Mode     string `yaml:"mode"`
}

type ConfigDev struct {
	Domain      string `yaml:"domain"`
	HttpTarget  string `yaml:"http_target"`
	HttpsTarget string `yaml:"https_target"`
	DnsTarget   string `yaml:"dns_target"`
}

type Config struct {
	file         string
	Domains      []string         `yaml:"domains"`
	ExternalIp   ConfigExternalIp `yaml:"external_ip"`
	ExternalIpv6 ConfigExternalIp `yaml:"external_ipv6"`
	Servers      []ConfigServer   `yaml:"servers"`
	Dev          ConfigDev        `yaml:"dev"`
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
