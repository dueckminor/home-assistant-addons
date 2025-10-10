package gateway

import (
	"context"
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/dueckminor/home-assistant-addons/go/dns"
	"github.com/gin-gonic/gin"
)

type DnsEndpoints struct {
	server dns.Server
	config *Config
}

func NewDns(server dns.Server) *DnsEndpoints {
	return &DnsEndpoints{
		server: server,
	}
}

func (d *DnsEndpoints) setupEndpoints(r *gin.RouterGroup) {
	dnsGroup := r.Group("/dns")
	dnsGroup.GET("/external/ipv4", d.GET_ExternalIpv4)
	dnsGroup.POST("/external/ipv4", d.POST_ExternalIpv4)
	dnsGroup.GET("/external/ipv6", d.GET_ExternalIpv6)
	dnsGroup.POST("/external/ipv6", d.POST_ExternalIpv6)

	dnsGroup.GET("/ipv4", d.GET_Ipv4)
	dnsGroup.GET("/ipv6", d.GET_Ipv6)
	dnsGroup.GET("/lookup", d.GET_DnsLookup)
}

func (d *DnsEndpoints) GET_ExternalIpv4(c *gin.Context) {
	addr, err := d.lookup(c.Request.Context(), d.config.ExternalIp.Options, false)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	c.JSON(200, gin.H{"method": d.config.ExternalIp.Source, "source": d.config.ExternalIp.Options, "address": addr, "timestamp": time.Now()})
}

func (d *DnsEndpoints) POST_ExternalIpv4(c *gin.Context) {
	var h gin.H
	c.BindJSON(&h)
	d.config.ExternalIp.Source = "dns"
	d.config.ExternalIp.Options = h["source"].(string)

	extIPv4 := dns.NewExternalIP("ip4", d.config.ExternalIp.Options)
	ip := extIPv4.ExternalIP()
	d.server.SetExternalIP(extIPv4)
	d.config.save()

	c.JSON(200, gin.H{"method": d.config.ExternalIp.Source, "source": d.config.ExternalIp.Options, "address": ip.String(), "timestamp": time.Now()})
}
func (d *DnsEndpoints) GET_ExternalIpv6(c *gin.Context) {
	addr, err := d.lookup(c.Request.Context(), d.config.ExternalIpv6.Options, true)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	c.JSON(200, gin.H{"method": d.config.ExternalIpv6.Source, "source": d.config.ExternalIpv6.Options, "address": addr, "timestamp": time.Now()})
}
func (d *DnsEndpoints) POST_ExternalIpv6(c *gin.Context) {
	var h gin.H
	c.BindJSON(&h)
	d.config.ExternalIp.Source = "dns"
	d.config.ExternalIpv6.Options = h["source"].(string)
	extIPv6 := dns.NewExternalIP("ip6", d.config.ExternalIpv6.Options)
	ip := extIPv6.ExternalIP()
	d.server.SetExternalIPv6(extIPv6)
	d.config.save()

	c.JSON(200, gin.H{"method": d.config.ExternalIpv6.Source, "source": d.config.ExternalIpv6.Options, "address": ip.String(), "timestamp": time.Now()})
}

func (d *DnsEndpoints) GET_Ipv4(c *gin.Context) {
	hostname := c.Query("hostname")
	if hostname == "" {
		c.JSON(400, gin.H{"error": "hostname parameter is required"})
		return
	}

	ips, err := net.LookupIP(hostname)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}

	for _, ip := range ips {
		if ip.To4() != nil { // IPv4
			c.JSON(200, gin.H{"ip": ip.String(), "timestamp": time.Now()})
			return
		}
	}
	c.JSON(404, gin.H{"error": "No IPv4 found"})
}

func (d *DnsEndpoints) GET_Ipv6(c *gin.Context) {
	hostname := c.Query("hostname")
	if hostname == "" {
		c.JSON(400, gin.H{"error": "hostname parameter is required"})
		return
	}

	ips, err := net.LookupIP(hostname)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}

	for _, ip := range ips {
		if ip.To4() == nil { // IPv6
			c.JSON(200, gin.H{"ip": ip.String(), "timestamp": time.Now()})
			return
		}
	}
	c.JSON(404, gin.H{"error": "No IPv6 found"})
}

func (d *DnsEndpoints) lookup(ctx context.Context, address string, ipv6 bool) (resolved string, err error) {
	ips, err := net.LookupIP(address)
	if err != nil {
		return "", err
	}

	for _, ip := range ips {
		if (ip.To4() == nil) == ipv6 { // IPv6
			return ip.String(), nil
		}
	}
	return "", fmt.Errorf("failed to resolve %s", address)
}

// GET_DnsLookup performs DNS lookups for various record types
func (d *DnsEndpoints) GET_DnsLookup(c *gin.Context) {
	hostname := c.Query("hostname")
	recordType := c.Query("type")

	// Create DNS client with 5 second timeout
	dnsClient := dns.NewDNSClient(5 * time.Second)

	// Perform the DNS lookup
	result := dnsClient.LookupDNS(hostname, recordType)

	// Handle errors
	if result.Error != "" {
		if hostname == "" {
			c.JSON(400, gin.H{"error": result.Error})
		} else if strings.Contains(result.Error, "Unsupported record type") {
			c.JSON(400, gin.H{"error": result.Error})
		} else if strings.Contains(result.Error, "No records found") || strings.Contains(result.Error, "No DNS response received") {
			c.JSON(404, gin.H{"error": result.Error, "type": result.Type})
		} else {
			c.JSON(500, gin.H{"error": result.Error})
		}
		return
	}

	// Return successful result
	c.JSON(200, result)
}
