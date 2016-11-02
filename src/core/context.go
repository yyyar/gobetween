/**
 * context.go - proxy context
 *
 * @author Yaroslav Pogrebnyak <yyyaroslav@gmail.com>
 */

package core

import (
	"net"
)

type Context interface {
	String() string
}

/**
 * Proxy tcp context
 */
type TcpContext struct {

	/**
	 * Current client connection
	 */
	Conn net.Conn
}

func (t TcpContext) String() string {
	return t.Conn.RemoteAddr().String()
}

/**
 * Proxy udp context
 */
type UdpContext struct {

	/**
	 * Current client remote address
	 */
	RemoteAddr net.UDPAddr
}

func (u UdpContext) String() string {
	return u.RemoteAddr.String()
}
