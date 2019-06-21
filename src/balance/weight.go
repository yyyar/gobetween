package balance

/**
 * weight.go - weight balance impl
 *
 * @author Yaroslav Pogrebnyak <yyyaroslav@gmail.com>
 */

import (
	"errors"
	"math/rand"

	"github.com/yyyar/gobetween/core"
	"github.com/yyyar/gobetween/logging"
)

/**
 * Weight balancer
 */
type WeightBalancer struct{}

var log = logging.For("balance/weight")

/**
 * Elect backend based on weight with priority strategy.
 * See https://tools.ietf.org/html/rfc2782, Priority and Weight sections
 */
func (b *WeightBalancer) Elect(context core.Context, backends []*core.Backend) (*core.Backend, error) {

	if len(backends) == 0 {
		return nil, errors.New("Can't elect backend, Backends empty")
	}

	// according to RFC we should use backends with lowest priority
	minPriority := backends[0].Priority
	// group of backends with priority == minPriority
	group := make([]*core.Backend, 0, len(backends))
	// sum of weights in the group
	groupSumWeight := 0

	// first pass: find lowest numbered priority and a group of backeds with it
	for _, backend := range backends {

		if backend.Priority > minPriority {
			continue
		}

		if backend.Priority < 0 {
			log.Warn("Ignoring invalid backend priority %v, should not be less than 0", backend.Priority)
			continue
		}

		if backend.Weight < 0 {
			log.Warn("Ignoring invalid backend weight %v, should not be less than 0", backend.Weight)
			continue
		}

		// got new lower (accroding to RFC, lower values are preferred) priority, reset
		if backend.Priority < minPriority {
			minPriority = backend.Priority
			group = make([]*core.Backend, 0, len(backends))
			groupSumWeight = 0
		}

		group = append(group, backend)
		groupSumWeight += backend.Weight
	}

	// corner case #1 -- group of just one backend, simply return
	if len(group) == 1 {
		return group[0], nil
	}

	// corner case #2 -- group of backends with weight 0 (allowed by RFC, but not handled by weight distribution algorithm)
	if groupSumWeight == 0 {
		return group[rand.Intn(len(group))], nil
	}

	r := rand.Intn(groupSumWeight)
	pos := 0

	// weight selection algorithm
	for _, backend := range group {
		pos += backend.Weight
		if r >= pos {
			continue
		}
		return backend, nil
	}

	return nil, errors.New("Can't elect backend")
}
