package dns

import (
	"net"
	"strings"
	"time"

	miekgdns "github.com/miekg/dns"
)

// DNSRecord represents a DNS record with all relevant information
type DNSRecord struct {
	Name     string  `json:"name"`
	Type     string  `json:"type"`
	TTL      uint32  `json:"ttl"`
	Value    any     `json:"value"`
	Priority *uint16 `json:"priority,omitempty"`
	Weight   *uint16 `json:"weight,omitempty"`
	Port     *uint16 `json:"port,omitempty"`
	Flag     *uint8  `json:"flag,omitempty"`
	Tag      *string `json:"tag,omitempty"`
}

// DNSLookupResult represents the result of a DNS lookup
type DNSLookupResult struct {
	Hostname  string      `json:"hostname"`
	Type      string      `json:"type"`
	Records   []DNSRecord `json:"records"`
	Timestamp time.Time   `json:"timestamp"`
	Error     string      `json:"error,omitempty"`
}

// DNSClient provides DNS lookup functionality
type DNSClient struct {
	timeout time.Duration
}

// NewDNSClient creates a new DNS client with the specified timeout
func NewDNSClient(timeout time.Duration) *DNSClient {
	return &DNSClient{
		timeout: timeout,
	}
}

// getSystemDNSServers retrieves the system's configured DNS servers
func (dc *DNSClient) getSystemDNSServers() []string {
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
func (dc *DNSClient) findAuthoritativeServers(hostname string, client *miekgdns.Client) []string {
	var servers []string

	// Split domain into parts to find the zone
	parts := strings.Split(strings.TrimSuffix(hostname, "."), ".")

	// Try from most specific to least specific (e.g., sub.example.com -> example.com -> com)
	for i := range parts {
		zoneName := strings.Join(parts[i:], ".")

		// Query for NS records of this zone
		msg := new(miekgdns.Msg)
		msg.SetQuestion(miekgdns.Fqdn(zoneName), miekgdns.TypeNS)

		// Try system DNS servers first, then fallback to public DNS
		systemServers := dc.getSystemDNSServers()
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

// parseDNSRecord converts a miekg/dns RR to our DNSRecord struct
func (dc *DNSClient) parseDNSRecord(rr miekgdns.RR) DNSRecord {
	record := DNSRecord{
		Name: rr.Header().Name,
		Type: miekgdns.TypeToString[rr.Header().Rrtype],
		TTL:  rr.Header().Ttl,
	}

	switch r := rr.(type) {
	case *miekgdns.A:
		record.Value = r.A.String()
	case *miekgdns.AAAA:
		record.Value = r.AAAA.String()
	case *miekgdns.NS:
		record.Value = r.Ns
	case *miekgdns.CNAME:
		record.Value = r.Target
	case *miekgdns.MX:
		record.Value = r.Mx
		priority := r.Preference
		record.Priority = &priority
	case *miekgdns.TXT:
		record.Value = strings.Join(r.Txt, " ")
	case *miekgdns.SOA:
		record.Value = map[string]any{
			"ns":      r.Ns,
			"mbox":    r.Mbox,
			"serial":  r.Serial,
			"refresh": r.Refresh,
			"retry":   r.Retry,
			"expire":  r.Expire,
			"minttl":  r.Minttl,
		}
	case *miekgdns.PTR:
		record.Value = r.Ptr
	case *miekgdns.SRV:
		record.Value = r.Target
		priority := r.Priority
		weight := r.Weight
		port := r.Port
		record.Priority = &priority
		record.Weight = &weight
		record.Port = &port
	case *miekgdns.CAA:
		record.Value = r.Value
		flag := r.Flag
		tag := r.Tag
		record.Flag = &flag
		record.Tag = &tag
	default:
		record.Value = rr.String()
	}

	return record
}

// LookupDNS performs DNS lookups for various record types
func (dc *DNSClient) LookupDNS(hostname, recordType string) *DNSLookupResult {
	result := &DNSLookupResult{
		Hostname:  hostname,
		Type:      recordType,
		Timestamp: time.Now(),
	}

	if hostname == "" {
		result.Error = "hostname parameter is required"
		return result
	}

	if recordType == "" {
		recordType = "A" // Default to A records
	}
	recordType = strings.ToUpper(recordType)

	// Create DNS client
	client := new(miekgdns.Client)
	client.Timeout = dc.timeout

	// Special handling for .local domains (mDNS)
	if strings.HasSuffix(hostname, ".local") {
		// For .local domains, use net.LookupIP as fallback since it supports mDNS
		if recordType == "A" || recordType == "AAAA" {
			ips, err := net.LookupIP(hostname)
			if err == nil && len(ips) > 0 {
				for _, ip := range ips {
					if (recordType == "A" && ip.To4() != nil) || (recordType == "AAAA" && ip.To4() == nil) {
						result.Records = append(result.Records, DNSRecord{
							Name:  hostname + ".",
							Type:  recordType,
							TTL:   300, // Default TTL for mDNS
							Value: ip.String(),
						})
					}
				}
				if len(result.Records) > 0 {
					return result
				}
			}
		}
	}

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
		result.Error = "Unsupported record type. Supported types: A, AAAA, NS, CNAME, MX, TXT, SOA, PTR, SRV, CAA"
		return result
	}

	msg.SetQuestion(miekgdns.Fqdn(hostname), qtype)

	// Build list of DNS servers to try
	dnsServers := []string{}

	// First, try to find authoritative nameservers for this domain
	authServers := dc.findAuthoritativeServers(hostname, client)
	dnsServers = append(dnsServers, authServers...)

	// Fallback to system DNS servers, then public DNS servers
	systemServers := dc.getSystemDNSServers()
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
		result.Error = "DNS lookup failed"
		if lastError != nil {
			result.Error += ": " + lastError.Error()
		}
		return result
	}

	if response == nil {
		result.Error = "No DNS response received"
		return result
	}

	// Check both Answer and Authority sections
	allRecords := append(response.Answer, response.Ns...)

	if len(allRecords) == 0 {
		result.Error = "No records found"
		return result
	}

	// Parse all records
	for _, rr := range allRecords {
		record := dc.parseDNSRecord(rr)
		result.Records = append(result.Records, record)
	}

	return result
}
