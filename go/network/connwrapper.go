package network

import (
	"fmt"
	"io"
	"net"
	"sync"
	"time"
)

type connWrapper struct {
	conn       net.Conn
	cacheRead  bool
	buff       []byte
	readAhead  []byte
	clientAddr net.Addr
}

func (w *connWrapper) HandleProxyProtocol() error {
	if w.conn == nil {
		return nil
	}

	proxyData := make([]byte, 4096)

	n, err := w.ReadAhead(proxyData[:6])
	if err != nil {
		return err
	}
	if n == 6 && string(proxyData[0:6]) == "PROXY " {
		// Read until \r\n
		totalRead := n
		for {
			if totalRead >= len(proxyData) {
				return nil // header too large
			}
			n, err := w.ReadAhead(proxyData[totalRead : totalRead+1])
			if err != nil {
				return err
			}
			totalRead += n
			if totalRead >= 2 && proxyData[totalRead-2] == '\r' && proxyData[totalRead-1] == '\n' {
				break
			}
		}
		w.readAhead = nil

		header := string(proxyData[:totalRead])
		var proto, srcAddr, destAddr string
		var srcPort, destPort int
		_, err = fmt.Sscanf(header, "PROXY %s %s %s %d %d\r\n", &proto, &srcAddr, &destAddr, &srcPort, &destPort)
		if err != nil {
			return err
		}
		ip := net.ParseIP(srcAddr)
		if ip == nil {
			return nil
		}

		w.clientAddr = &net.TCPAddr{
			IP:   ip,
			Port: srcPort,
		}
	}
	return nil
}

func (w *connWrapper) ReadAhead(b []byte) (n int, err error) {
	if w.conn == nil {
		return 0, nil
	}
	n, err = w.conn.Read(b)
	if n > 0 {
		w.readAhead = append(w.readAhead, b[0:n]...)
	}
	return n, err
}

func (w *connWrapper) Read(b []byte) (n int, err error) {
	if w.conn == nil {
		return 0, nil
	}
	if len(w.readAhead) > 0 {
		n = copy(b, w.readAhead)
		if w.cacheRead && n > 0 {
			w.buff = append(w.buff, b[0:n]...)
		}
		w.readAhead = w.readAhead[n:]
		return n, nil
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
	if w.clientAddr != nil {
		return w.clientAddr
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
	wg := sync.WaitGroup{}
	wg.Add(2)

	go func() {
		defer wg.Done()
		// when the server closes the connection,
		// it's no longer necessary to send something
		// -> lets close the client connection
		defer client.Close()

		io.Copy(client, server) // nolint: errcheck
	}()

	go func() {
		defer wg.Done()
		io.Copy(server, client) // nolint: errcheck
	}()

	wg.Wait()
}
