package balance

/**
 * ipporthash1.go - semi-consistent iphash balance impl also using the source prot
 *
 * Patched by Elias Weingärtner (elias.weingaertner@cumulocity.com)
 */

import (
	"errors"
	"hash/fnv"
	"strconv"
	"github.com/yyyar/gobetween/core"
)

/**
 * Iphash balancer
 */
type IpPorthash1Balancer struct {
}

/**
 * Elect backend using semi-consistent iphash strategy. This is naive implementation
 * using Key+Node Hash Algorithm for stable sharding described at http://kennethxu.blogspot.com/2012/11/sharding-algorithm.html
 * It survives removing nodes (removing stability), so that clients connected to backends that have not been removed stay
 * untouched.


 *
 */
func (b *IpPorthash1Balancer) Elect(context core.Context, backends []*core.Backend) (*core.Backend, error) {

	if len(backends) == 0 {
		return nil, errors.New("Can't elect backend, Backends empty")
	}

	var result *core.Backend
	{
		var bestHash uint32

		for i, backend := range backends {
			hasher := fnv.New32a()
			hasher.Write(context.Ip())

			portbytes := []byte(strconv.Itoa(context.Port()))
			hasher.Write(portbytes)

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
