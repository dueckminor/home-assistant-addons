package network

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"

	"github.com/gin-gonic/gin"
)

////////////////////////////////////////////////////////////////////////////////

type dialFunc func() (net.Conn, error)

func (d dialFunc) DialCtx(ctx context.Context, sni string) (net.Conn, error) {
	return d()
}
func NewDialTCPRaw(network string, addr string) DialCtx {
	return (dialFunc)(func() (net.Conn, error) {
		return net.Dial(network, addr)
	})
}

////////////////////////////////////////////////////////////////////////////////

func NewProxyDial(dialer DialCtx) ProxyDialCtx {
	return proxyDial{dialer: dialer}
}

type proxyDial struct {
	dialer DialCtx
}

func (pd proxyDial) ProxyDialCtx(ctx context.Context, client net.Conn, sni string) (net.Conn, error) {
	conn, err := pd.dialer.DialCtx(ctx, sni)
	if err != nil {
		return nil, err
	}

	proxyHeader := buildProxyProtocolHeader(client, conn)
	_, err = conn.Write(proxyHeader)
	if err != nil {
		conn.Close()
		return nil, err
	}

	return conn, nil
}

func buildProxyProtocolHeader(clientConn, targetConn net.Conn) []byte {
	clientAddr := clientConn.RemoteAddr().(*net.TCPAddr)
	targetAddr := targetConn.RemoteAddr().(*net.TCPAddr)

	var protocol string
	if clientAddr.IP.To4() != nil {
		protocol = "TCP4"
	} else {
		protocol = "TCP6"
	}

	header := fmt.Sprintf("PROXY %s %s %s %d %d\r\n",
		protocol,
		clientAddr.IP.String(),
		targetAddr.IP.String(),
		clientAddr.Port,
		targetAddr.Port)

	return []byte(header)
}

////////////////////////////////////////////////////////////////////////////////

type dialFixedAddress func() (net.Conn, error)

func (f dialFixedAddress) ServeCtx(ctx context.Context, conn net.Conn) {
	defer conn.Close()
	targetConn, err := f()
	if err != nil {
		fmt.Println("Dial Err:", err)
		return
	}
	forwardConnect(conn, targetConn)
}

////////////////////////////////////////////////////////////////////////////////

func NewServeTLS(address string) ServeCtx {
	tlsConfig := &tls.Config{
		ServerName: address,
	}
	return (dialFixedAddress)(func() (net.Conn, error) {
		return tls.Dial("tcp", address, tlsConfig)
	})
}

func NewServeTLSInsecure(serverName string) ServeCtx {
	tlsConfig := &tls.Config{
		InsecureSkipVerify: true,
		ServerName:         serverName,
	}
	return (dialFixedAddress)(func() (net.Conn, error) {
		return tls.Dial("tcp", serverName, tlsConfig)
	})
}

////////////////////////////////////////////////////////////////////////////////

func NewServeTCP(address string) ServeCtx {
	return (dialFixedAddress)(func() (net.Conn, error) {
		return net.Dial("tcp", address)
	})
}

////////////////////////////////////////////////////////////////////////////////

func NewGinHandler(r *gin.Engine) ServeCtx {
	l := MakeListener()
	go r.RunListener(l)
	return l
}
