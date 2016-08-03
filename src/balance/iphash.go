/**
 * iphash.go - iphash balance impl
 *
 * @author Yaroslav Pogrebnyak <yyyaroslav@gmail.com>
 */

package balance

import (
	"../core"
	"errors"
	"math"
)

/**
 * Iphash balancer
 */
type IphashBalancer struct{}

/**
 * Elect backend using iphash strategy
 * It's naive impl (most possibly with bad performance) using
 * FNV-1 hash (https://en.wikipedia.org/wiki/Fowler%E2%80%93Noll%E2%80%93Vo_hash_function)
 * TODO: Improve as needed
 */
func (b *IphashBalancer) Elect(context *core.Context, backends []core.Backend) (*core.Backend, error) {

	if len(backends) == 0 {
		return nil, errors.New("Can't elect backend, Backends empty")
	}

	// TODO: Replace using byte IP addr instead of string

	ip := (*context).String()

	hash := 11
	for c := range ip {
		hash = (hash * c) ^ 13
	}

	hash = int(math.Floor(math.Mod(float64(hash), float64(len(backends)))))

	backend := backends[hash]

	return &backend, nil
}
