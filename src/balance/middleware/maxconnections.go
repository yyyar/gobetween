package middleware

/**
 * maxconn.go - max connections middleware
 */

import (
	"errors"

	"github.com/yyyar/gobetween/core"
	"github.com/yyyar/gobetween/logging"
)

/**
 * MaxConnectionsMiddleware middleware
 * Filters out backends that have reached their max_connections limit
 */
type MaxConnectionsMiddleware struct {
	Delegate core.Balancer
}

/**
 * Elect backend filtering out backends that have reached max connections
 */
func (b *MaxConnectionsMiddleware) Elect(ctx core.Context, backends []*core.Backend) (*core.Backend, error) {
	log := logging.For("balance/middleware/maxconn")

	eligible := make([]*core.Backend, 0, len(backends))

	for _, backend := range backends {
		// Skip backends that have reached their connection limit
		if backend.MaxConnections > 0 && backend.Stats.ActiveConnections >= uint(backend.MaxConnections) {
			log.Debug("Backend ", backend.Address(), " excluded: active connections (",
				backend.Stats.ActiveConnections, ") >= max_connections (", backend.MaxConnections, ")")
			continue
		}
		eligible = append(eligible, backend)
	}

	if len(eligible) == 0 {
		return nil, errors.New("all backends have reached max connections limit")
	}

	return b.Delegate.Elect(ctx, eligible)
}
