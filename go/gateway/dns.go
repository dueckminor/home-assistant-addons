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

// getSystemDNSServers retrieves the system's configured DNS servers
func (d *DnsEndpoints) getSystemDNSServers() []string {
	var servers []string

	// Read system DNS configuration
	config, err := miekgdns.ClientConfigFromFile("/etc/resolv.conf")
	if err == nil && len(config.Servers) > 0 {
		for _, server := range config.Servers {
			// Add port 53 if not specified
			if !strings.Contains(server, ":") {
				server += ":53"
			}
			servers = append(servers, server)
		}
	}

	// Fallback: try to get system nameservers via net package
	if len(servers) == 0 {
		// On some systems, we can try looking at common locations
		// This is a basic fallback - the miekg/dns method above should work on most systems
		servers = append(servers, "127.0.0.53:53") // systemd-resolved common address
	}

	return servers
}

// findAuthoritativeServers dynamically discovers authoritative nameservers for a domain
func (d *DnsEndpoints) findAuthoritativeServers(hostname string, client *miekgdns.Client) []string {
	var servers []string

	// Split domain into parts to find the zone
	parts := strings.Split(strings.TrimSuffix(hostname, "."), ".")

	// Try from most specific to least specific (e.g., sub.example.com -> example.com -> com)
	for i := 0; i < len(parts); i++ {
		zoneName := strings.Join(parts[i:], ".")

		// Query for NS records of this zone
		msg := new(miekgdns.Msg)
		msg.SetQuestion(miekgdns.Fqdn(zoneName), miekgdns.TypeNS)

		// Try system DNS servers first, then fallback to public DNS
		systemServers := d.getSystemDNSServers()
		allServers := append(systemServers, "8.8.8.8:53", "1.1.1.1:53")

		for _, server := range allServers {
			response, _, err := client.Exchange(msg, server)
			if err != nil {
				continue
			}

			// Look for NS records in both Answer and Authority sections
			allRecords := append(response.Answer, response.Ns...)

			for _, record := range allRecords {
				if nsRecord, ok := record.(*miekgdns.NS); ok {
					nsServer := strings.TrimSuffix(nsRecord.Ns, ".")

					// Resolve the NS server to IP address
					ips, err := net.LookupIP(nsServer)
					if err == nil && len(ips) > 0 {
						// Use the first IPv4 address
						for _, ip := range ips {
							if ip.To4() != nil {
								servers = append(servers, ip.String()+":53")
								break
							}
						}
					}

					// Also add the hostname:53 as fallback
					servers = append(servers, nsServer+":53")
				}
			}

			// If we found NS records, stop looking at higher levels
			if len(servers) > 0 {
				return servers
			}
		}
	}

	return servers
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

	// Build list of DNS servers to try
	dnsServers := []string{}

	// First, try to find authoritative nameservers for this domain
	authServers := d.findAuthoritativeServers(hostname, client)
	dnsServers = append(dnsServers, authServers...)

	// Fallback to system DNS servers, then public DNS servers
	systemServers := d.getSystemDNSServers()
	dnsServers = append(dnsServers, systemServers...)
	dnsServers = append(dnsServers, "8.8.8.8:53", "1.1.1.1:53")

	var response *miekgdns.Msg
	var err error
	var lastError error

	// Try multiple DNS servers
	for _, server := range dnsServers {
		response, _, err = client.Exchange(msg, server)
		if err == nil && response != nil && (len(response.Answer) > 0 || len(response.Ns) > 0) {
			break
		}
		lastError = err
	}

	if err != nil {
		errorMsg := "DNS lookup failed"
		if lastError != nil {
			errorMsg += ": " + lastError.Error()
		}
		c.JSON(500, gin.H{"error": errorMsg})
		return
	}

	if response == nil {
		c.JSON(404, gin.H{"error": "No DNS response received", "type": recordType})
		return
	}

	// Parse responses based on record type
	var records []map[string]interface{}

	// Check both Answer and Authority sections
	allRecords := append(response.Answer, response.Ns...)

	if len(allRecords) == 0 {
		c.JSON(404, gin.H{"error": "No records found", "type": recordType})
		return
	}

	for _, ans := range allRecords {
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
