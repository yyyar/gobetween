/**
 * weight.go - weight balance impl
 *
 * @author Yaroslav Pogrebnyak <yyyaroslav@gmail.com>
 */

package balance

import (
	"../config"
	"../core"
	"errors"
	"math/rand"
)

/**
 * Priority balancer
 */
type PriorityBalancer struct{}

/**
 * Constructor
 */
func NewPriorityBalancer(cfg config.BalanceConfig) interface{} {
	return &PriorityBalancer{}
}

/**
 * Elect backend based on weight strategy
 */
func (b *PriorityBalancer) Elect(context core.Context, backends []*core.Backend) (*core.Backend, error) {

	if len(backends) == 0 {
		return nil, errors.New("Can't elect backend, Backends empty")
	}

	/* count the backends with the same priority */
	matchingPriority := 0
	bestPriority := 0
	for _, backend := range backends {
		if (matchingPriority == 0 || backend.Priority < bestPriority) {
			bestPriority = backend.Priority
			matchingPriority = 1
		} else if backend.Priority == bestPriority  {
			bestPriority = backend.Priority
			matchingPriority ++;
		}

	}

	r := rand.Intn(matchingPriority)
	pos := 0

	for _, backend := range backends {
		if backend.Priority == bestPriority  {
			if (pos == r) {
				return backend, nil
			}
			pos ++
		}
	}

	return nil, errors.New("Can't elect backend")
}