/**
 * priority.go - priority balance implementation
 * select backends with lowest priority, then randomly choose one based on 'weight'
 *
 * @author quedunk <quedunk@gmail.com>
 */

package balance

import (
	"errors"
	"math/rand"

	"github.com/yyyar/gobetween/core"
)

/**
 * Weight balancer
 */
type PriorityBalancer struct{}

/**
 * Elect backend based on weight strategy
 */
func (b *PriorityBalancer) Elect(context core.Context, backends []*core.Backend) (*core.Backend, error) {

	if len(backends) == 0 {
		return nil, errors.New("Can't elect backend, Backends empty")
	}

	/* count the total weight of all backends with the same priority */
	matchingTotalWeight := 0
	bestPriority := 0
	for _, backend := range backends {
		if backend.Weight <= 0 {
			return nil, errors.New("Invalid backend weight 0")
		}
		if matchingTotalWeight == 0 || backend.Priority < bestPriority {
			// if matchingTotalWeight is 0, its our first backend in the loop
			// otherwise, we've found a backend with a better priority, so start our weight calc again
			bestPriority = backend.Priority
			matchingTotalWeight = backend.Weight
		} else if backend.Priority == bestPriority {
			matchingTotalWeight += backend.Weight
		}

	}

	// rand.Intn(100) returns a random int n, 0 <= n < 100
	// if we discovered a single server with weight 1, we want r=1
	r := rand.Intn(matchingTotalWeight) + 1
	pos := 0

	for _, backend := range backends {
		if backend.Priority == bestPriority {
			pos += backend.Weight
			// if the added weight equals or exceeds the random value, its our guy
			if pos >= r {
				return backend, nil
			}
		}
	}

	return nil, errors.New("Can't elect backend")
}
