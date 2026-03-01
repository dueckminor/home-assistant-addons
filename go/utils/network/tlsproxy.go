package network

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
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

type ProxyDialCtx interface {
	ProxyDialCtx(ctx context.Context, client net.Conn, sni string) (net.Conn, error)
}

type wrapDialCtx struct {
	dialer DialCtx
}

func (wd *wrapDialCtx) ProxyDialCtx(ctx context.Context, client net.Conn, sni string) (net.Conn, error) {
	return wd.dialer.DialCtx(ctx, sni)
}

type TLSProxy interface {
	io.Closer
	SetExternalIp(address string)
	AddHandler(sni string, handler any)
	SetMetricCallback(metricCallback MetricCallback)
	DeleteHandler(sni string)
	InternalOnly(sni string)
	AddTLSCertificates(sni string, tlsCertificates []tls.Certificate)
	EnableProxyProtocol(enable bool)
}
type tlsProxy struct {
	listener       net.Listener
	httpsListener  Listener
	httpsServer    *http.Server
	httpHandlers   map[string]http.Handler
	dialHandlers   map[string]ProxyDialCtx
	tlsConfigs     map[string]*tls.Config
	internal       map[string]bool
	externalAddr   net.IP
	metricCallback MetricCallback
	proxyProtocol  bool
}

func NewTLSProxy(network string, address string) (TLSProxy, error) {
	tp := &tlsProxy{
		httpHandlers: make(map[string]http.Handler),
		dialHandlers: make(map[string]ProxyDialCtx),
		tlsConfigs:   make(map[string]*tls.Config),
		internal:     make(map[string]bool),
	}
	err := tp.start(network, address)
	if err != nil {
		return nil, err
	}

	return tp, nil
}

func (tp *tlsProxy) EnableProxyProtocol(enable bool) {
	tp.proxyProtocol = enable
}

func (tp *tlsProxy) SetMetricCallback(metricCallback MetricCallback) {
	tp.metricCallback = metricCallback
}

func (tp *tlsProxy) Close() error {
	if tp.listener != nil {
		tp.listener.Close()
		tp.listener = nil
	}
	if tp.httpsServer != nil {
		close(tp.httpsListener)
		tp.httpsListener = nil
		tp.httpsServer.Close()
		tp.httpsServer = nil
	}
	return nil
}

func (tp *tlsProxy) start(network string, address string) error {
	tp.startHTTPSServer()

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
	var serveHandler http.Handler
	var dialHandler ProxyDialCtx

	switch v := handler.(type) {
	case http.Handler:
		serveHandler = v
	case ProxyDialCtx:
		dialHandler = v
	// wrap DialCtx into ProxyDialCtx
	case DialCtx:
		dialHandler = &wrapDialCtx{dialer: v}
	default:
		return
	}
	if serveHandler != nil {
		tp.httpHandlers[sni] = serveHandler
		delete(tp.dialHandlers, sni)
	} else {
		tp.dialHandlers[sni] = dialHandler
		delete(tp.httpHandlers, sni)
	}
}

func (tp *tlsProxy) DeleteHandler(sni string) {
	delete(tp.dialHandlers, sni)
	delete(tp.httpHandlers, sni)
	delete(tp.internal, sni)
}

func (tp *tlsProxy) InternalOnly(sni string) {
	tp.internal[sni] = true
}

func (tp *tlsProxy) AddTLSCertificates(sni string, tlsCertificates []tls.Certificate) {
	if len(tlsCertificates) == 0 {
		delete(tp.tlsConfigs, sni)
		return
	}
	tlsConfig := &tls.Config{
		Certificates: tlsCertificates,
		NextProtos:   []string{"h2", "http/1.1"},
		MinVersion:   tls.VersionTLS12,
		CipherSuites: []uint16{
			// TLS 1.3 cipher suites (order doesn't matter for TLS 1.3)
			tls.TLS_AES_128_GCM_SHA256,
			tls.TLS_AES_256_GCM_SHA384,
			tls.TLS_CHACHA20_POLY1305_SHA256,
			// TLS 1.2 cipher suites (in preference order)
			tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,
			tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
			tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305_SHA256,
			tls.TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305_SHA256,
		},
		CurvePreferences: []tls.CurveID{
			tls.X25519MLKEM768,
			tls.X25519,
			tls.CurveP256,
			tls.CurveP384,
		},
	}
	tp.tlsConfigs[sni] = tlsConfig
}

func (tp *tlsProxy) getHandler(sni string) (httpHandler http.Handler, dial ProxyDialCtx, internal bool) {
	if !tp.isValidHostname(sni) {
		return nil, nil, false
	}
	wildcard := "*." + strings.Join(strings.Split(sni, ".")[1:], ".")
	internal = tp.internal[sni] || tp.internal[wildcard]
	var ok bool
	if httpHandler, ok = tp.httpHandlers[sni]; !ok {
		if dial, ok = tp.dialHandlers[sni]; !ok {
			if httpHandler, ok = tp.httpHandlers[wildcard]; !ok {
				dial, ok = tp.dialHandlers[wildcard]
			}
		}
	}
	return httpHandler, dial, internal
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
	closeConn := true
	defer func() {
		if closeConn {
			conn.Close()
		}
	}()

	var err error
	if tp.proxyProtocol {
		conn, err = HandleProxyProtocol(conn)
		if err != nil {
			fmt.Println("Proxy Protocol Err:", err)
			return
		}
	}

	clientAddr := conn.RemoteAddr()

	clientHello, conn := ReadTlsClientHello(conn)
	sni := clientHello.ServerName

	tlsConfig := tp.getTLSConfig(sni)
	httpHandler, dial, _ := tp.getHandler(sni)

	if nil == dial && (tlsConfig == nil || nil == httpHandler) {
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

	fmt.Println("ServerName:", sni)

	if httpHandler != nil {
		closeConn = false
		tp.httpsListener.ServeCtx(ctx, conn)
		return
	}

	targetConn, err := dial.ProxyDialCtx(ctx, conn, sni)
	if err != nil {
		fmt.Println("Dial Err:", err)
		return
	}
	forwardConnect(conn, targetConn)
}

func (tp *tlsProxy) startHTTPSServer() {
	tp.httpsServer = &http.Server{
		TLSConfig: &tls.Config{GetConfigForClient: func(clientHelloInfo *tls.ClientHelloInfo) (*tls.Config, error) {
			sni := clientHelloInfo.ServerName
			tlsConfig := tp.getTLSConfig(sni)
			if tlsConfig == nil {
				return nil, os.ErrInvalid
			}
			return tlsConfig, nil
		}},
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			sni := r.Host
			httpHandler, _, _ := tp.getHandler(sni)
			if httpHandler == nil {
				http.NotFound(w, r)
				return
			}
			httpHandler.ServeHTTP(w, r)
		}),
	}
	tp.httpsListener = MakeListener()
	go tp.httpsServer.ServeTLS(tp.httpsListener, "", "")
}

func ReadTlsClientHello(conn net.Conn) (result *tls.ClientHelloInfo, conn2 net.Conn) {
	bufferedConn := ConnBufferRead(conn)

	tlsConn := tls.Server(ConnWithWriteDevNull(bufferedConn), &tls.Config{GetConfigForClient: func(clientHelloInfo *tls.ClientHelloInfo) (*tls.Config, error) {
		result = clientHelloInfo
		return nil, os.ErrInvalid
	}})
	_ = tlsConn.Handshake()

	return result, bufferedConn.GetConnReadAgain()
}
