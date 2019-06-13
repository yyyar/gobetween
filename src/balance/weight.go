package balance

/**
 * weight.go - weight balance impl
 *
 * @author Yaroslav Pogrebnyak <yyyaroslav@gmail.com>
 */

import (
	"errors"
	"fmt"
	"math/rand"

	"github.com/yyyar/gobetween/core"
)

/**
 * Weight balancer
 */
type WeightBalancer struct{}

/**
 * Elect backend based on weight with priority strategy.
 * See https://tools.ietf.org/html/rfc2782, Priority and Weight sections
 */
func (b *WeightBalancer) Elect(context core.Context, backends []*core.Backend) (*core.Backend, error) {

	if len(backends) == 0 {
		return nil, errors.New("Can't elect backend, Backends empty")
	}

	if len(backends) == 1 {
		return backends[0], nil
	}

	// according to RFC we should use backends with lowest priority
	minPriority := backends[0].Priority
	// group of backends with priority == minPriority
	group := make([]*core.Backend, 0, len(backends))
	// sum of weights in the group
	groupSumWeight := 0

	for _, backend := range backends {

		if backend.Priority > minPriority {
			continue
		}

		if backend.Priority < 0 {
			return nil, fmt.Errorf("Invalid backend priority, shold not be less than 0: %v", backend.Priority)
		}

		if backend.Weight <= 0 {
			return nil, fmt.Errorf("Invalid backend weight, should not be less or equal to 0: %v", backend.Weight)
		}

		// got new lower priority, reset
		if backend.Priority < minPriority {
			minPriority = backend.Priority
			group = make([]*core.Backend, 0, len(backends))
			groupSumWeight = 0
		}

		group = append(group, backend)
		groupSumWeight += backend.Weight
	}

	if len(group) == 1 {
		return group[0], nil
	}

	r := rand.Intn(groupSumWeight)
	pos := 0

	for _, backend := range group {
		pos += backend.Weight
		if r >= pos {
			continue
		}
		return backend, nil
	}

	return nil, errors.New("Can't elect backend")
}
