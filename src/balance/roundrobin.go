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

	sorted := make([]*core.Backend, len(backends))
	copy(sorted, backends)

	sort.SliceStable(sorted, func(i, j int) bool {
		return sorted[i].Target.String() < sorted[j].Target.String()
	})

	if b.current >= len(sorted) {
		b.current = 0
	}

	backend := sorted[b.current]
	b.current += 1

	return backend, nil
}
