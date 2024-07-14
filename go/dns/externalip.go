package dns

import (
	"fmt"
	"net"
	"time"
)

type ExternalIP interface {
	ExternalIP() net.IP
}

func NewExternalIP(network string, address string) ExternalIP {
	return &externalIP{
		network: network,
		address: address,
	}
}

type externalIP struct {
	ip      net.IP
	ttl     time.Time
	network string
	address string
}

func (e *externalIP) ExternalIP() net.IP {
	if len(e.ip) > 0 && time.Now().Before(e.ttl) {
		return e.ip
	}
	e.refresh()
	return e.ip
}

func (e *externalIP) refresh() {
	fmt.Println("try to resolve", e.network, e.address)
	addr, err := net.ResolveIPAddr(e.network, e.address)
	if err != nil {
		fmt.Println("failed:", err)
		return
	}
	e.ip = addr.IP
	e.ttl = time.Now().Add(time.Second * 30)
}
