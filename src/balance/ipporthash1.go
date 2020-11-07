package balance

/**
 * ipporthash.go - semi-consistent ip-porthash balance impl
 *
 * Based on iphash1.go.
 */

import (
	"errors"
	"hash/fnv"
	"strconv"

	"github.com/yyyar/gobetween/core"
)

/**
 * IpPorthash1 balancer
 */
type IpPorthash1Balancer struct {
}

/**
 * Elect backend using semi-consistent ip+port hash strategy.
 * This is useful to balance connections coming from a NATed source.
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
			ipPort := context.Ip().String() + strconv.Itoa(context.Port())
			hasher.Write([]byte(ipPort))
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
