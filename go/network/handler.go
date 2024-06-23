package network

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
)

////////////////////////////////////////////////////////////////////////////////

type dialFunc func() (net.Conn, error)

func (d dialFunc) Dial(sni string) (net.Conn, error) {
	return d()
}
func (d dialFunc) DialCtx(ctx context.Context, sni string) (net.Conn, error) {
	return d()
}
func NewDialTCPRaw(network string, addr string) DialCtx {
	return (dialFunc)(func() (net.Conn, error) {
		return net.Dial(network, addr)
	})
}

////////////////////////////////////////////////////////////////////////////////

type dialFixedAddress func() (net.Conn, error)

func (f dialFixedAddress) Serve(conn net.Conn) {
	f.ServeCtx(context.Background(), conn)
}

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
