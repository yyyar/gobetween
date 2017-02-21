/**
 * extract.go - extractor of hostname from ClientHello
 *
 * @author Illarion Kovalchuk <illarion.kovalchuk@gmail.com>
 */

package sni

import (
	"../../../logging"
	"bytes"
	"crypto/tls"
	"io"
	"net"
	"reflect"
	"time"
)

type bufferConn struct {
	io.Reader
}

type localAddr struct{}

func (l localAddr) String() string {
	return "127.0.0.1"
}

func (l localAddr) Network() string {
	return "tcp"
}

func newBufferConn(b []byte) *bufferConn {
	return &bufferConn{bytes.NewReader(b)}
}

func (c bufferConn) Write(b []byte) (n int, err error) {
	return 0, nil
}

func (c bufferConn) Close() error {
	return nil
}

func (c bufferConn) LocalAddr() net.Addr {
	return localAddr{}
}

func (c bufferConn) RemoteAddr() net.Addr {
	return localAddr{}
}

func (c bufferConn) SetDeadline(t time.Time) error {
	return nil
}

func (c bufferConn) SetReadDeadline(t time.Time) error {
	return nil
}

func (c bufferConn) SetWriteDeadline(t time.Time) error {
	return nil
}

func extractHostname(buf []byte) (result string) {
	conn := tls.Server(newBufferConn(buf), &tls.Config{})
	defer conn.Close()

	conn.Handshake()
	result = conn.ConnectionState().ServerName

	if result != "" {
		return
	}

	defer func() {
		if r := recover(); r != nil {
			if err, ok := r.(error); ok {
				logging.Error("Was not able to extract hostname from ClientHello due to :", err)
			}
			result = ""
		}
	}()

	// Prior to go1.8 ConnectionState.ServerName will be not filled, so we'll try to get it from reflection
	p := reflect.ValueOf(conn)
	v := reflect.Indirect(p)

	result = v.FieldByName("serverName").String()
	return
}
