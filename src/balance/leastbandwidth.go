package balance

/**
 * leastbandwidth.go - leastbandwidth balance impl
 *
 * @author Yaroslav Pogrebnyak <yyyaroslav@gmail.com>
 */

import (
	"errors"

	"github.com/yyyar/gobetween/core"
)

/**
 * Leastbandwidth balancer
 */
type LeastbandwidthBalancer struct{}

/**
 * Elect backend using leastbandwidth strategy
 */
func (b *LeastbandwidthBalancer) Elect(context core.Context, backends []*core.Backend) (*core.Backend, error) {

	if len(backends) == 0 {
		return nil, errors.New("Can't elect backend, Backends empty")
	}

	least := backends[0]
	for _, b := range backends {
		if b.Stats.TxSecond+b.Stats.RxSecond < least.Stats.TxSecond+least.Stats.RxSecond {
			least = b
		}
	}

	return least, nil
}
