package dns

import (
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/miekg/dns"
)

type Server interface {
	SetExternalIP(externalIP ExternalIP) error
	SetExternalIPv6(externalIP ExternalIP) error
	AddDomains(domains ...string) error
	AddProxyDomain(domain string, target string) error
	SetChallenge(domain string, challenge string) error
}

type domain struct {
	name         string
	challenge    string
	proxy_target string
}

func (d *domain) makeNS() dns.RR {
	return &dns.NS{
		Hdr: dns.RR_Header{
			Name:   fmt.Sprintf("%s.", d.name),
			Rrtype: dns.TypeNS,
			Class:  dns.ClassINET,
			Ttl:    60,
		},
		Ns: fmt.Sprintf("ns1.%s.", d.name),
	}
}

func (d *domain) makeSOA() dns.RR {
	return &dns.SOA{
		Hdr: dns.RR_Header{
			Name:   fmt.Sprintf("%s.", d.name),
			Rrtype: dns.TypeSOA,
			Class:  dns.ClassINET,
			Ttl:    60,
		},
		Ns:      fmt.Sprintf("ns1.%s.", d.name),
		Mbox:    fmt.Sprintf("admin.%s.", d.name),
		Serial:  uint32(time.Now().Unix()),
		Refresh: 28800,
		Retry:   7200,
		Expire:  600,
		Minttl:  60,
	}
}

type domainWithHost struct {
	domain
	query string
	host  string
}

func (d *domainWithHost) makeA(ip net.IP) dns.RR {
	return &dns.A{
		Hdr: dns.RR_Header{
			Name:   d.query,
			Rrtype: dns.TypeA,
			Class:  dns.ClassINET,
			Ttl:    60,
		},
		A: ip,
	}
}
func (d *domainWithHost) makeAAAA(ip net.IP) dns.RR {
	return &dns.AAAA{
		Hdr: dns.RR_Header{
			Name:   d.query,
			Rrtype: dns.TypeAAAA,
			Class:  dns.ClassINET,
			Ttl:    60,
		},
		AAAA: ip,
	}
}

func (d *domainWithHost) makeCNAME(cname string) dns.RR {
	return &dns.CNAME{
		Hdr: dns.RR_Header{
			Name:   d.query,
			Rrtype: dns.TypeCNAME,
			Class:  dns.ClassINET,
			Ttl:    60,
		},
		Target: cname + ".",
	}
}

func (d *domainWithHost) makeACME() dns.RR {
	return &dns.TXT{
		Hdr: dns.RR_Header{
			Name:   d.query,
			Rrtype: dns.TypeTXT,
			Class:  dns.ClassINET,
			Ttl:    10,
		},
		Txt: []string{d.challenge},
	}
}

type server struct {
	mux *dns.ServeMux
	udp *dns.Server
	tcp *dns.Server

	domains []*domain

	ipv4 ExternalIP
	ipv6 ExternalIP
}

func (s *server) SetExternalIP(externalIP ExternalIP) error {
	s.ipv4 = externalIP
	return nil
}

func (s *server) SetExternalIPv6(externalIP ExternalIP) error {
	s.ipv6 = externalIP
	return nil
}

func (s *server) getDomain(name string) *domain {
	for _, domain := range s.domains {
		if domain.name == name {
			return domain
		}
	}
	return nil
}

func (s *server) AddDomains(domains ...string) error {
	for _, name := range domains {
		d := s.getDomain(name)
		if d == nil {
			d = &domain{
				name: name,
			}
			s.domains = append(s.domains, d)
		}
	}
	return nil
}

func (s *server) AddProxyDomain(name string, target string) error {
	d := s.getDomain(name)
	if d == nil {
		d = &domain{
			name: name,
		}
		s.domains = append(s.domains, d)
	}
	d.proxy_target = target
	return nil
}

func (s *server) SetChallenge(domain string, challenge string) error {
	d := s.getDomain(domain)
	if d == nil {
		return nil
	}
	d.challenge = challenge
	return nil
}

func NewServer(addr string) (s Server, err error) {
	server := &server{}

	server.mux = dns.NewServeMux()
	server.mux.HandleFunc(".", server.dnsHandleFunc)

	server.udp = &dns.Server{
		Addr:    addr,
		Net:     "udp",
		Handler: server.mux,
	}
	server.tcp = &dns.Server{
		Addr:    addr,
		Net:     "tcp",
		Handler: server.mux,
	}

	go func() {
		err := server.udp.ListenAndServe()
		if err != nil {
			panic(err)
		}
	}()
	go func() {
		err := server.tcp.ListenAndServe()
		if err != nil {
			panic(err)
		}
	}()

	return server, nil
}

func (s *server) Close() error {
	s.tcp.Shutdown()
	return s.udp.Shutdown()
}

func (s *server) questionToHostAndDomain(q dns.Question) (dwh *domainWithHost) {
	name := strings.ToLower(q.Name)
	if name == "" {
		return nil
	}
	name = name[:len(name)-1]

	for _, d := range s.domains {
		if name == d.name {
			return &domainWithHost{
				domain: *d,
				query:  q.Name,
				host:   "",
			}
		} else if strings.HasSuffix(name, "."+d.name) {
			return &domainWithHost{
				domain: *d,
				query:  q.Name,
				host:   name[:len(name)-len(d.name)-1],
			}
		}
	}
	return nil
}

func (s *server) dnsHandleFunc(w dns.ResponseWriter, r *dns.Msg) {

	m := new(dns.Msg)
	m.SetReply(r)

	fmt.Println("DNS: Remote-Addr:", w.RemoteAddr())

	d := s.questionToHostAndDomain(r.Question[0])

	if d != nil && d.proxy_target != "" {
		dnsClient := new(dns.Client)
		target := d.proxy_target
		target = strings.TrimPrefix(target, "udp://")
		if strings.HasPrefix(target, "tcp://") {
			target = target[6:]
			dnsClient.Net = "tcp"
		}
		resp, runtime, err := dnsClient.Exchange(r, target)
		fmt.Println("runtime:", runtime)
		fmt.Println("err:", err)
		if resp != nil {
			w.WriteMsg(resp)
		} else {
			w.WriteMsg(m)
		}
		return
	}

	if d != nil {
		switch r.Question[0].Qtype {
		case dns.TypeTXT:
			fmt.Println("host:", d.host)
			if d.host == "_acme-challenge" {
				m.Answer = append(m.Answer, d.makeACME())
				m.Ns = append(m.Ns, d.makeNS())
			}
		case dns.TypeCNAME:
			if d.host != "" {
				m.Ns = append(m.Ns, d.makeNS())
			}
		case dns.TypeA:
			if d.host != "" {
				addr := s.ipv4.ExternalIP()
				if len(addr) == 4 {
					m.Answer = append(m.Answer, d.makeA(addr))
				}
				m.Ns = append(m.Ns, d.makeNS())
			}
		case dns.TypeAAAA:
			addr := s.ipv6.ExternalIP()
			if len(addr) == 16 {
				m.Answer = append(m.Answer, d.makeAAAA(addr))
			}
			m.Ns = append(m.Ns, d.makeNS())
		default:
			fmt.Println("?", r.Question[0].Name)
		}
		if len(m.Answer) == 0 {
			m.Ns = append(m.Ns, d.makeSOA())
		}
	}

	if r.IsTsig() != nil {
		if w.TsigStatus() == nil {
			m.SetTsig(r.Extra[len(r.Extra)-1].(*dns.TSIG).Hdr.Name, dns.HmacSHA256, 300, time.Now().Unix())
		} else {
			println("Status", w.TsigStatus().Error())
		}
	}

	fmt.Printf("<<<<<<<<<<\n%v>>>>>>>>>>\n", m.String())

	err := w.WriteMsg(m)
	if err != nil {
		fmt.Println("failed to write DNS responses:", err)
	}
}
