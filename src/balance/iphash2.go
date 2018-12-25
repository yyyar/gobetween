/**
 * iphash.go - iphash2 balance implementation
 *
 * @author Yaroslav Pogrebnyak <yyyaroslav@gmail.com>
 */

package balance

import (
	"errors"
	"github.com/yyyar/gobetween/config"
	"github.com/yyyar/gobetween/core"
)

/**
 * Iphash2 balancer
 */
type Iphash2Balancer struct {
	cfg config.IpHash2BalanceConfig
}

/**
 * Constructor
 */
func NewIphash2Balancer(cfg config.BalanceConfig) interface{} {
	return &Iphash2Balancer{
		*cfg.IpHash2BalanceConfig,
	}
}

/**
 * Elect backend using iphash strategy
 * This balancer is stable in both adding and removing backends
 * It keeps mapping cache for some period of time.
 */
func (b *Iphash2Balancer) Elect(context core.Context, backends []*core.Backend) (*core.Backend, error) {

	if len(backends) == 0 {
		return nil, errors.New("Can't elect backend, Backends empty")
	}

	// TODO: Add implementation

	return backends[0], nil
}
