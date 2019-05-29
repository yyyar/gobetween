package balance

/**
 * leastconn.go - leastconn balance impl
 *
 * @author Yaroslav Pogrebnyak <yyyaroslav@gmail.com>
 */

import (
	"errors"

	"github.com/yyyar/gobetween/core"
)

/**
 * Leastconn balancer
 */
type LeastconnBalancer struct{}

/**
 * Elect backend using roundrobin strategy
 */
func (b *LeastconnBalancer) Elect(context core.Context, backends []*core.Backend) (*core.Backend, error) {

	if len(backends) == 0 {
		return nil, errors.New("Can't elect backend, Backends empty")
	}

	least := backends[0]

	for key, backend := range backends {
		if backend.Stats.ActiveConnections <= least.Stats.ActiveConnections {
			least = backends[key]
		}
	}

	return least, nil
}
