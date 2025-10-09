package gateway

import (
	"context"
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/dueckminor/home-assistant-addons/go/dns"
	"github.com/gin-gonic/gin"
	miekgdns "github.com/miekg/dns"
)

type DnsEndpoints struct {
	server dns.Server

	externalIpv4 *ConfigExternalIp
	externalIpv6 *ConfigExternalIp
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
	addr, err := d.lookup(c.Request.Context(), d.externalIpv4.Options, false)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	c.JSON(200, gin.H{"method": d.externalIpv4.Source, "source": d.externalIpv4.Options, "address": addr, "timestamp": time.Now()})
}

func (d *DnsEndpoints) POST_ExternalIpv4(c *gin.Context) {
	var h gin.H
	c.BindJSON(&h)
	d.externalIpv4.Options = h["source"].(string)
}
func (d *DnsEndpoints) GET_ExternalIpv6(c *gin.Context) {
	addr, err := d.lookup(c.Request.Context(), d.externalIpv6.Options, true)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	c.JSON(200, gin.H{"method": d.externalIpv6.Source, "source": d.externalIpv6.Options, "address": addr, "timestamp": time.Now()})
}
func (d *DnsEndpoints) POST_ExternalIpv6(c *gin.Context) {
	var h gin.H
	c.BindJSON(&h)
	d.externalIpv6.Options = h["source"].(string)
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
	recordType := strings.ToUpper(c.Query("type"))

	if hostname == "" {
		c.JSON(400, gin.H{"error": "hostname parameter is required"})
		return
	}

	if recordType == "" {
		recordType = "A" // Default to A records
	}

	// Create DNS client
	client := new(miekgdns.Client)
	client.Timeout = 5 * time.Second

	// Create DNS message
	msg := new(miekgdns.Msg)

	var qtype uint16
	switch recordType {
	case "A":
		qtype = miekgdns.TypeA
	case "AAAA":
		qtype = miekgdns.TypeAAAA
	case "NS":
		qtype = miekgdns.TypeNS
	case "CNAME":
		qtype = miekgdns.TypeCNAME
	case "MX":
		qtype = miekgdns.TypeMX
	case "TXT":
		qtype = miekgdns.TypeTXT
	case "SOA":
		qtype = miekgdns.TypeSOA
	case "PTR":
		qtype = miekgdns.TypePTR
	case "SRV":
		qtype = miekgdns.TypeSRV
	case "CAA":
		qtype = miekgdns.TypeCAA
	default:
		c.JSON(400, gin.H{"error": "Unsupported record type. Supported types: A, AAAA, NS, CNAME, MX, TXT, SOA, PTR, SRV, CAA"})
		return
	}

	msg.SetQuestion(miekgdns.Fqdn(hostname), qtype)

	// Use system DNS servers or fallback to public DNS
	dnsServers := []string{"8.8.8.8:53", "1.1.1.1:53"}

	var response *miekgdns.Msg
	var err error

	// Try multiple DNS servers
	for _, server := range dnsServers {
		response, _, err = client.Exchange(msg, server)
		if err == nil && response != nil {
			break
		}
	}

	if err != nil {
		c.JSON(500, gin.H{"error": "DNS lookup failed: " + err.Error()})
		return
	}

	if response == nil || len(response.Answer) == 0 {
		c.JSON(404, gin.H{"error": "No records found", "type": recordType})
		return
	}

	// Parse responses based on record type
	var records []map[string]interface{}

	for _, ans := range response.Answer {
		record := map[string]interface{}{
			"name": ans.Header().Name,
			"type": miekgdns.TypeToString[ans.Header().Rrtype],
			"ttl":  ans.Header().Ttl,
		}

		switch rr := ans.(type) {
		case *miekgdns.A:
			record["value"] = rr.A.String()
		case *miekgdns.AAAA:
			record["value"] = rr.AAAA.String()
		case *miekgdns.NS:
			record["value"] = rr.Ns
		case *miekgdns.CNAME:
			record["value"] = rr.Target
		case *miekgdns.MX:
			record["value"] = rr.Mx
			record["priority"] = rr.Preference
		case *miekgdns.TXT:
			record["value"] = strings.Join(rr.Txt, " ")
		case *miekgdns.SOA:
			record["value"] = map[string]interface{}{
				"ns":      rr.Ns,
				"mbox":    rr.Mbox,
				"serial":  rr.Serial,
				"refresh": rr.Refresh,
				"retry":   rr.Retry,
				"expire":  rr.Expire,
				"minttl":  rr.Minttl,
			}
		case *miekgdns.PTR:
			record["value"] = rr.Ptr
		case *miekgdns.SRV:
			record["value"] = rr.Target
			record["priority"] = rr.Priority
			record["weight"] = rr.Weight
			record["port"] = rr.Port
		case *miekgdns.CAA:
			record["value"] = rr.Value
			record["flag"] = rr.Flag
			record["tag"] = rr.Tag
		default:
			record["value"] = ans.String()
		}

		records = append(records, record)
	}

	c.JSON(200, gin.H{
		"hostname":  hostname,
		"type":      recordType,
		"records":   records,
		"timestamp": time.Now(),
	})
}
