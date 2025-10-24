package dns

import (
	"fmt"
	"net"
)

type ExternalIP interface {
	ExternalIP() net.IP
	Refresh() (net.IP, error)
}

func NewExternalIP(network string, address string) ExternalIP {
	return &externalIP{
		network: network,
		address: address,
	}
}

type externalIP struct {
	ip      net.IP
	network string
	address string
}

func (e *externalIP) ExternalIP() net.IP {
	return e.ip
}

func (e *externalIP) Refresh() (net.IP, error) {
	addr, err := net.ResolveIPAddr(e.network, e.address)
	if err != nil {
		fmt.Println("failed:", err)
		return nil, err
	}
	e.ip = addr.IP
	return e.ip, nil
}
