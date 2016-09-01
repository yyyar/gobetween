/**
 * server.go - server
 *
 * @author Illarion Kovalchuk
 * @author Yaroslav Pogrebnyak <yyyaroslav@gmail.com>
 */

package core

import (
	"../config"
)

/**
 * Server interface
 */
type Server interface {

	/**
	 * Start server
	 */
	Start() error

	/**
	 * Stop server and wait until it stop
	 */
	Stop()

	/**
	 * Get server configuration
	 */
	Cfg() config.Server
}
