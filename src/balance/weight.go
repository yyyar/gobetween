/**
 * weight.go - weight balance impl
 *
 * @author Yaroslav Pogrebnyak <yyyaroslav@gmail.com>
 */

package balance

import (
	"errors"
	"math/rand"

	"../core"
)

/**
 * Weight balancer
 */
type WeightBalancer struct{}

/**
 * Elect backend based on weight strategy
 * TODO: Ensure backends are sorted in the same way (not it's not bacause of map in scheduler)
 */
func (b *WeightBalancer) Elect(context core.Context, backends []*core.Backend) (*core.Backend, error) {

	if len(backends) == 0 {
		return nil, errors.New("Can't elect backend, Backends empty")
	}

	totalWeight := 0
	for _, backend := range backends {
		if backend.Weight <= 0 {
			return nil, errors.New("Invalid backend weight 0")
		}
		totalWeight += backend.Weight
	}

	r := rand.Intn(totalWeight)
	pos := 0

	for _, backend := range backends {
		pos += backend.Weight
		if r >= pos {
			continue
		}
		return backend, nil
	}

	return nil, errors.New("Cant elect backend")
}
