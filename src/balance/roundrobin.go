package balance

/**
 * roundrobin.go - roundrobin balance impl
 *
 * @author Yaroslav Pogrebnyak <yyyaroslav@gmail.com>
 */

import (
	"errors"
	"sort"

	"github.com/yyyar/gobetween/core"
)

/**
 * Roundrobin balancer
 */
type RoundrobinBalancer struct {

	/* Current backend position */
	current int
}

/**
 * Elect backend using roundrobin strategy
 */
func (b *RoundrobinBalancer) Elect(context core.Context, backends []*core.Backend) (*core.Backend, error) {

	if len(backends) == 0 {
		return nil, errors.New("Can't elect backend, Backends empty")
	}

	sort.SliceStable(backends, func(i, j int) bool {
		return backends[i].Target.String() < backends[j].Target.String()
	})

	if b.current >= len(backends) {
		b.current = 0
	}

	backend := backends[b.current]
	b.current += 1

	return backend, nil
}
