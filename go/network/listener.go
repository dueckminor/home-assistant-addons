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

// methods of net.Listener

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

// methods of network.ServeCtx

func (l Listener) ServeCtx(ctx context.Context, conn net.Conn) {
	// no need to do this here: defer conn.Close()
	// the connection will be closed by the caller of Accept
	select {
	case <-ctx.Done():
		return
	case l <- conn:
		return
	}
}

////////////////////////////////////////////////////////////////////////////////

func MakeListener() Listener {
	return make(Listener)
}
