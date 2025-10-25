package homeassistant

import (
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
)

type ExternalIP interface {
	ExternalIP() net.IP
	Refresh() (net.IP, error)
}

var supervisorURI = "http://supervisor"
var supervisorToken = ""

func init() {
	uri := os.Getenv("SUPERVISOR_URI")
	if uri != "" {
		supervisorURI = uri
	}
	supervisorToken = os.Getenv("SUPERVISOR_TOKEN")
}

func NewExternalIP() ExternalIP {
	return &externalIP{
		token: supervisorToken,
	}
}

type externalIP struct {
	token string
	ip    net.IP
}

func (e *externalIP) ExternalIP() net.IP {
	return e.ip
}

func (e *externalIP) Refresh() (net.IP, error) {
	// use the http://supervisor/network/info endpoint to get the external IP
	req, err := http.NewRequest("GET", supervisorURI+"/network/info", nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+e.token)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var networkInfo struct {
		Data struct {
			Interfaces []struct {
				IPv6 struct {
					Address []string `json:"address"`
				} `json:"ipv6"`
			} `json:"interfaces"`
		} `json:"data"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&networkInfo); err != nil {
		return nil, err
	}

	for _, iface := range networkInfo.Data.Interfaces {
		for _, addr := range iface.IPv6.Address {
			if ip, _, err := net.ParseCIDR(addr); err == nil && ip.To4() == nil {
				e.ip = ip
				return ip, nil
			}
		}
	}

	return nil, fmt.Errorf("no valid IPv6 address found")
}
