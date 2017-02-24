/**
 * iphash.go - iphash balance impl
 *
 * @author Yaroslav Pogrebnyak <yyyaroslav@gmail.com>
 */

package balance

import (
	"errors"
	"math"

	"../core"
)

/**
 * Iphash balancer
 */
type IphashBalancer struct{}

/**
 * Elect backend using iphash strategy
 * It's naive impl (most possibly with bad performance) using
 * FNV-1a hash (https://en.wikipedia.org/wiki/Fowler%E2%80%93Noll%E2%80%93Vo_hash_function)
 * TODO: Improve as needed
 */
func (b *IphashBalancer) Elect(context core.Context, backends []*core.Backend) (*core.Backend, error) {

	if len(backends) == 0 {
		return nil, errors.New("Can't elect backend, Backends empty")
	}

	ip := context.Ip()

	hash := 11
	for _, b := range ip {
		hash = (hash ^ int(b)) * 13
	}

	hash = int(math.Floor(math.Mod(float64(hash), float64(len(backends)))))

	backend := backends[hash]

	return backend, nil
}
