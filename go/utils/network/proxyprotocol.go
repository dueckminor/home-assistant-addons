package network

import (
	"fmt"
	"net"
)

func HandleProxyProtocol(conn net.Conn) (net.Conn, error) {
	proxyData := make([]byte, 4096)
	n, err := conn.Read(proxyData[:6])
	if err != nil {
		return nil, err
	}
	if n == 6 && string(proxyData[0:6]) == "PROXY " {
		// Read until \r\n
		totalRead := n
		for {
			if totalRead >= len(proxyData) {
				return nil, nil // header too large
			}
			n, err := conn.Read(proxyData[totalRead : totalRead+1])
			if err != nil {
				return nil, err
			}
			totalRead += n
			if totalRead >= 2 && proxyData[totalRead-2] == '\r' && proxyData[totalRead-1] == '\n' {
				break
			}
		}

		header := string(proxyData[:totalRead])
		var proto, srcAddr, destAddr string
		var srcPort, destPort int
		_, err = fmt.Sscanf(header, "PROXY %s %s %s %d %d\r\n", &proto, &srcAddr, &destAddr, &srcPort, &destPort)
		if err != nil {
			return nil, err
		}
		ip := net.ParseIP(srcAddr)
		if ip == nil {
			return conn, nil
		}
		return ConnWithRemoteAddr(conn, &net.TCPAddr{IP: ip, Port: srcPort}), nil
	}
	return ConnWithReadPrefix(conn, proxyData[:n]), nil
}
