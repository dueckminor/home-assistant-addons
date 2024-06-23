package network

import "net"

type Listener chan net.Conn

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

func MakeListener() Listener {
	return make(chan net.Conn)
}
