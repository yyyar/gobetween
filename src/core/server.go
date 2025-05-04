package core

/**
 * server.go - server
 *
 * @author Illarion Kovalchuk
 * @author Yaroslav Pogrebnyak <yyyaroslav@gmail.com>
 */

import (
	"github.com/yyyar/gobetween/config"
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
	 * UpdateBackends allows you to set a new backend config
	 */
	UpdateBackends(backends *[]Backend)

	/**
	 * Get server configuration
	 */
	Cfg() config.Server
}
