package gateway

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
	Domains      []string         `yaml:"domains"`
	ExternalIp   ConfigExternalIp `yaml:"external_ip"`
	ExternalIpv6 ConfigExternalIp `yaml:"external_ipv6"`
	Servers      []ConfigServer   `yaml:"servers"`
	Dev          ConfigDev        `yaml:"dev"`
}
