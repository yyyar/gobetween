/**
 * context.go - proxy context
 *
 * @author Yaroslav Pogrebnyak <yyyaroslav@gmail.com>
 */

package core

import (
	"net"
)

/**
 * Proxy context
 */
type Context struct {

	/**
	 * Current client connection
	 */
	Conn net.Conn
}
