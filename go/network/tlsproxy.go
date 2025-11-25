package network

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"strings"
	"time"
)

type ServeCtx interface {
	ServeCtx(ctx context.Context, conn net.Conn)
}
type DialCtx interface {
	DialCtx(ctx context.Context, sni string) (net.Conn, error)
}

type TLSProxy interface {
	io.Closer
	SetExternalIp(address string)
	AddHandler(sni string, handler any)
	SetMetricCallback(metricCallback MetricCallback)
	DeleteHandler(sni string)
	InternalOnly(sni string)
	AddTLSConfig(sni string, tlsConfig *tls.Config)
}

type tlsProxy struct {
	listener       net.Listener
	serveHandlers  map[string]ServeCtx
	dialHandlers   map[string]DialCtx
	tlsConfigs     map[string]*tls.Config
	internal       map[string]bool
	externalAddr   net.IP
	metricCallback MetricCallback
}

func NewTLSProxy(network string, address string) (TLSProxy, error) {
	tp := &tlsProxy{
		serveHandlers: make(map[string]ServeCtx),
		dialHandlers:  make(map[string]DialCtx),
		tlsConfigs:    make(map[string]*tls.Config),
		internal:      make(map[string]bool),
	}
	err := tp.start(network, address)
	if err != nil {
		return nil, err
	}

	return tp, nil
}

func (tp *tlsProxy) SetMetricCallback(metricCallback MetricCallback) {
	tp.metricCallback = metricCallback
}

func (tp *tlsProxy) Close() error {
	if tp.listener != nil {
		tp.listener.Close()
		tp.listener = nil
	}
	return nil
}

func (tp *tlsProxy) start(network string, address string) error {
	listener, err := net.Listen(network, address)
	if err != nil {
		return err
	}
	go func() {

		for {
			conn, err := listener.Accept()
			if err != nil {
				log.Fatal("could not accept client connection", err)
			}
			go func() {
				remoteAddr := conn.RemoteAddr()
				fmt.Printf("client '%v' connected!\n", remoteAddr)
				tp.ServeCtx(context.Background(), conn)
				fmt.Printf("client '%v' disconnected!\n", remoteAddr)
			}()
		}
	}()
	return nil
}

func (tp *tlsProxy) SetExternalIp(address string) {
	tp.externalAddr = net.ParseIP(address)
}

func (tp *tlsProxy) AddHandler(sni string, handler any) {
	var serveHandler ServeCtx
	var dialHandler DialCtx

	switch v := handler.(type) {
	case ServeCtx:
		serveHandler = v
	case DialCtx:
		dialHandler = v
	default:
		return
	}
	if serveHandler != nil {
		tp.serveHandlers[sni] = serveHandler
		delete(tp.dialHandlers, sni)
	} else {
		tp.dialHandlers[sni] = dialHandler
		delete(tp.serveHandlers, sni)
	}
}

func (tp *tlsProxy) DeleteHandler(sni string) {
	delete(tp.dialHandlers, sni)
	delete(tp.serveHandlers, sni)
	delete(tp.internal, sni)
}

func (tp *tlsProxy) InternalOnly(sni string) {
	tp.internal[sni] = true
}

func (tp *tlsProxy) AddTLSConfig(sni string, tlsConfig *tls.Config) {
	tp.tlsConfigs[sni] = tlsConfig
}

func (tp *tlsProxy) getHandler(sni string) (serve ServeCtx, dial DialCtx, internal bool) {
	if !tp.isValidHostname(sni) {
		return nil, nil, false
	}
	wildcard := "*." + strings.Join(strings.Split(sni, ".")[1:], ".")
	internal = tp.internal[sni] || tp.internal[wildcard]
	var ok bool
	if serve, ok = tp.serveHandlers[sni]; !ok {
		if dial, ok = tp.dialHandlers[sni]; !ok {
			if serve, ok = tp.serveHandlers[wildcard]; !ok {
				dial, ok = tp.dialHandlers[wildcard]
			}
		}
	}
	return serve, dial, internal
}

func (tp *tlsProxy) getTLSConfig(sni string) *tls.Config {
	if !tp.isValidHostname(sni) {
		return nil
	}
	tlsConfig := tp.tlsConfigs[sni]
	if tlsConfig != nil {
		return tlsConfig
	}
	sni = "*." + strings.Join(strings.Split(sni, ".")[1:], ".")
	return tp.tlsConfigs[sni]
}

func (tp *tlsProxy) isValidHostname(sni string) bool {
	if sni == "" {
		return false
	}
	return sni[0] != '*'
}

// Serve reads the TLS client hello (and stores it in a cache)
// depending on the included sni header it chooses how to continue.
// The following options are available
//   - close the connection without sending something back to the client.
//     This prevents most port scanning attacks
//   - redirect the complete connection to a different host without
//     removing the TLS layer (here the cache is used to repeat the client hello)
//   - complete the TLS handshake and forward the content to a different host
func (tp *tlsProxy) ServeCtx(ctx context.Context, conn net.Conn) {
	clientWrapper := &connWrapper{conn: conn, cacheRead: true}

	clientAddr := conn.RemoteAddr()

	closeConn := true
	defer func() {
		if closeConn {
			conn.Close()
		}
	}()

	var sni string
	var serve ServeCtx
	var dial DialCtx
	var internal bool

	tlsConn := tls.Server(clientWrapper, &tls.Config{GetConfigForClient: func(clientHelloInfo *tls.ClientHelloInfo) (*tls.Config, error) {
		clientWrapper.cacheRead = false
		sni = clientHelloInfo.ServerName
		fmt.Println("ServerName:", sni)

		serve, dial, internal = tp.getHandler(sni)

		fmt.Println("Remote:", clientAddr)
		if internal {
			fmt.Println("Internal!")
		}

		if nil == serve && nil == dial {
			fmt.Println("-> dropped")
			return nil, os.ErrInvalid
		}
		if dial != nil {
			// from now on the connection is handled by dialer
			// disconnect conn from the clientWrapper, so that the tls
			// implementation can no longer use it
			clientWrapper.conn = nil
			// let the tls handshake fail
			return nil, os.ErrInvalid
		}

		tlsConfig := tp.getTLSConfig(sni)
		if tlsConfig == nil {
			fmt.Println("-> dropped (have no cert")
			// let the tls handshake fail
			return nil, os.ErrInvalid
		}

		return tlsConfig, nil
	}})

	err := tlsConn.Handshake()

	if nil == serve && nil == dial {
		fmt.Println("ServerName:", sni, "rejected")
		if tp.metricCallback != nil {
			tp.metricCallback(Metric{
				Timestamp:    time.Now(),
				ClientAddr:   clientAddr.String(),
				Hostname:     sni,
				ResponseCode: 666,
			})
		}
		return
	}

	if err != nil && dial == nil {
		fmt.Println("Handshake Err:", err)
		if tp.metricCallback != nil {
			tp.metricCallback(Metric{
				Timestamp:    time.Now(),
				ClientAddr:   clientAddr.String(),
				Hostname:     sni,
				ResponseCode: 667,
			})
		}
		return
	}

	fmt.Println("ServerName:", sni)

	if serve != nil {
		// from now on hostImpl is responsible to close the connection
		closeConn = false
		serve.ServeCtx(ctx, tlsConn)
		return
	}

	tp.handleDialer(ctx, sni, conn, dial, clientWrapper.buff)
}

func (tp *tlsProxy) handleDialer(ctx context.Context, sni string, conn net.Conn, dial DialCtx, clientHello []byte) {
	targetConn, err := dial.DialCtx(ctx, sni)
	if err != nil {
		fmt.Println("Dial Err:", err)
		return
	}

	_, err = targetConn.Write(clientHello)
	if err != nil {
		fmt.Println("Forwarding client hello failed:", err)
		return
	}

	forwardConnect(conn, targetConn)
}
