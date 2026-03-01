package network

import (
	"io"
	"net"
	"sync"
)

// /////////////////////////////////////////////////////////////////////////////

type connWrapper struct {
	net.Conn
}

func (c *connWrapper) Unwrap() net.Conn {
	return c.Conn
}

// /////////////////////////////////////////////////////////////////////////////

func ConnWithRemoteAddr(conn net.Conn, remoteAddr net.Addr) net.Conn {
	return &connWithRemoteAddr{connWrapper: connWrapper{conn}, remoteAddr: remoteAddr}
}

type connWithRemoteAddr struct {
	connWrapper
	remoteAddr net.Addr
}

func (c *connWithRemoteAddr) RemoteAddr() net.Addr {
	return c.remoteAddr
}

// /////////////////////////////////////////////////////////////////////////////

func ConnWithWriteDevNull(conn net.Conn) net.Conn {
	return &connWithWriteDevNull{connWrapper: connWrapper{conn}}
}

type connWithWriteDevNull struct {
	connWrapper
}

func (c *connWithWriteDevNull) Write(b []byte) (n int, err error) {
	return len(b), nil
}

// /////////////////////////////////////////////////////////////////////////////

func ConnWithReadPrefix(conn net.Conn, prefix []byte) net.Conn {
	if len(prefix) == 0 {
		return conn
	}
	if existingReadPrefix, ok := findConnWithReadPrefix(conn); ok {
		// Our connection is already a ConnWithReadPrefix,
		// so we can just prepend our buffer to the existing prefix
		existingReadPrefix.prefix = append(prefix, existingReadPrefix.prefix...)
		return conn
	}
	return &connWithReadPrefix{connWrapper: connWrapper{conn}, prefix: prefix}
}

type connWithReadPrefix struct {
	connWrapper
	prefix []byte
}

func (c *connWithReadPrefix) Read(b []byte) (n int, err error) {
	if len(c.prefix) > 0 {
		n = copy(b, c.prefix)
		c.prefix = c.prefix[n:]
		return n, nil
	}
	return c.Conn.Read(b)
}

func findConnWithReadPrefix(conn net.Conn) (*connWithReadPrefix, bool) {
	for conn != nil {
		if c, ok := conn.(*connWithReadPrefix); ok {
			return c, true
		}
		wrapper, ok := conn.(interface{ Unwrap() net.Conn })
		if !ok {
			break
		}
		conn = wrapper.Unwrap()
	}
	return nil, false
}

// /////////////////////////////////////////////////////////////////////////////

func ConnBufferRead(conn net.Conn) *connBufferRead {
	return &connBufferRead{connWrapper: connWrapper{conn}, buf: make([]byte, 0)}
}

type connBufferRead struct {
	connWrapper
	buf []byte
}

func (c *connBufferRead) Read(b []byte) (n int, err error) {
	n, err = c.Conn.Read(b)
	if n > 0 {
		c.buf = append(c.buf, b[0:n]...)
	}
	return n, err
}

func (c *connBufferRead) GetConnReadAgain() net.Conn {
	return ConnWithReadPrefix(c.Conn, c.buf)
}
func (c *connBufferRead) GetConnSkipAllRead() net.Conn {
	return c.Conn
}
func (c *connBufferRead) GetConnSkipFirstN(n int) net.Conn {
	if n <= 0 {
		return ConnWithReadPrefix(c.Conn, c.buf)
	}
	if n < len(c.buf) {
		return ConnWithReadPrefix(c.Conn, c.buf[n:])
	}
	return c.Conn
}

// /////////////////////////////////////////////////////////////////////////////

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
