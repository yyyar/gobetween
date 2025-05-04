package server

/**
 * server.go - server creator
 *
 * @author Illarion Kovalchuk
 * @author Yaroslav Pogrebnyak <yyyaroslav@gmail.com>
 */

import (
	"errors"

	"github.com/yyyar/gobetween/config"
	"github.com/yyyar/gobetween/core"
	"github.com/yyyar/gobetween/server/tcp"
	"github.com/yyyar/gobetween/server/udp"
)

/**
 * Creates new Server based on cfg.Protocol
 */
func New(name string, cfg config.Server, completed chan<- struct{}) (core.Server, error) {
	switch cfg.Protocol {
	case "tls", "tcp":
		return tcp.New(name, cfg, completed)
	case "udp":
		return udp.New(name, cfg, completed)
	default:
		return nil, errors.New("Can't create server for protocol " + cfg.Protocol)
	}
}
