package balance

/**
 * iphash1.go - semi-consistent iphash balance impl
 *
 * @author Illarion Kovalchuk <illarion.kovalchuk@gmail.com>
 */

import (
	"errors"
	"hash/fnv"

	"github.com/yyyar/gobetween/core"
)

/**
 * Iphash balancer
 */
type Iphash1Balancer struct {
}

/**
 * Elect backend using semi-consistent iphash strategy. This is naive implementation
 * using Key+Node Hash Algorithm for stable sharding described at http://kennethxu.blogspot.com/2012/11/sharding-algorithm.html
 * It survives removing nodes (removing stability), so that clients connected to backends that have not been removed stay
 * untouched.
 *
 */
func (b *Iphash1Balancer) Elect(context core.Context, backends []*core.Backend) (*core.Backend, error) {

	if len(backends) == 0 {
		return nil, errors.New("Can't elect backend, Backends empty")
	}

	var result *core.Backend
	{
		var bestHash uint32

		for i, backend := range backends {
			hasher := fnv.New32a()
			hasher.Write(context.Ip())
			hasher.Write([]byte(backend.Address()))
			s32 := hasher.Sum32()
			if s32 > bestHash {
				bestHash = s32
				result = backends[i]
			}
		}
	}

	return result, nil
}
