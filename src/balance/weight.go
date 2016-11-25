/**
 * weight.go - weight balance impl
 *
 * @author Yaroslav Pogrebnyak <yyyaroslav@gmail.com>
 */

package balance

import (
	"../core"
	"errors"
	"math/rand"
)

/**
 * Weight balancer
 */
type WeightBalancer struct{}

/**
 * Elect backend based on weight strategy
 * TODO: Ensure backends are sorted in the same way (not it's not bacause of map in scheduler)
 */
func (b *WeightBalancer) Elect(context *core.Context, backends []core.Backend) (*core.Backend, error) {

	totalWeight := 0
	for _, backend := range backends {
		totalWeight += backend.Weight
	}

	r := rand.Intn(totalWeight)
	pos := 0

	for _, backend := range backends {
		pos += backend.Weight
		if r >= pos {
			continue
		}
		return &backend, nil
	}

	return nil, errors.New("Cant elect backend, or backends list is empty")
}
