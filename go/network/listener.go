package network

import (
	"context"
	"net"
)

// a Listener builds the bridge between a TLSProxy and gin.engine.RunListener
// by implementing the following interfaces:
// - net.Listener
// - network.ServeCtx
type Listener chan net.Conn

// net.Listener

func (l Listener) Accept() (net.Conn, error) {
	conn := <-l
	return conn, nil
}
func (l Listener) Close() error {
	return nil
}
func (l Listener) Addr() net.Addr {
	return nil
}

// network.ServeCtx

func (l Listener) Serve(conn net.Conn) {
	// no need to do this here: defer conn.Close()
	// the connection will be closed by the caller of Accept
	l <- conn
}
func (l Listener) ServeCtx(ctx context.Context, conn net.Conn) {
	// no need to do this here: defer conn.Close()
	// the connection will be closed by the caller of Accept
	l <- conn
}

////////////////////////////////////////////////////////////////////////////////

func MakeListener() Listener {
	return make(chan net.Conn)
}
