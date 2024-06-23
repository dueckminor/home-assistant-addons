package network

import (
	"io"
	"net"
	"time"
)

type connWrapper struct {
	conn      net.Conn
	cacheRead bool
	buff      []byte
}

func (w *connWrapper) Read(b []byte) (n int, err error) {
	if w.conn == nil {
		return 0, nil
	}
	n, err = w.conn.Read(b)
	if w.cacheRead && n > 0 {
		w.buff = append(w.buff, b[0:n]...)
	}
	return
}
func (w *connWrapper) Write(b []byte) (n int, err error) {
	if w.conn == nil {
		return len(b), nil
	}
	return w.conn.Write(b)
}
func (w *connWrapper) Close() error {
	if w.conn == nil {
		return nil
	}
	return w.conn.Close()
}
func (w *connWrapper) LocalAddr() net.Addr {
	if w.conn == nil {
		return nil
	}
	return w.conn.LocalAddr()
}
func (w *connWrapper) RemoteAddr() net.Addr {
	if w.conn == nil {
		return nil
	}
	return w.conn.RemoteAddr()
}
func (w *connWrapper) SetDeadline(t time.Time) error {
	if w.conn == nil {
		return nil
	}
	return w.conn.SetDeadline(t)
}
func (w *connWrapper) SetReadDeadline(t time.Time) error {
	if w.conn == nil {
		return nil
	}
	return w.conn.SetReadDeadline(t)
}
func (w *connWrapper) SetWriteDeadline(t time.Time) error {
	if w.conn == nil {
		return nil
	}
	return w.conn.SetWriteDeadline(t)
}

func forwardConnect(client, server net.Conn) {
	done := make(chan bool, 2)

	go func() {
		// when the server closes the connection,
		// it's no longer necessary to send something
		// -> lets close the client connection
		defer client.Close()

		io.Copy(client, server) // nolint: errcheck
		done <- true
	}()

	go func() {
		io.Copy(server, client) // nolint: errcheck
		done <- true
	}()

	<-done
	<-done
}
