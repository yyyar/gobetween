/**
 * weight.go - weight balance impl
 *
 * @author Yaroslav Pogrebnyak <yyyaroslav@gmail.com>
 */

package balance

import (
	"errors"
	"math/rand"
	"sort"

	"../core"
)

/**
 * Weight balancer
 */
type WeightBalancer struct{}

/**
 * Elect backend based on weight strategy
 */
func (b *WeightBalancer) Elect(context core.Context, backends []*core.Backend) (*core.Backend, error) {

	if len(backends) == 0 {
		return nil, errors.New("Can't elect backend, Backends empty")
	}

	sorted := make([]*core.Backend, len(backends))
	copy(sorted, backends)

	sort.SliceStable(sorted, func(i, j int) bool {
		return sorted[i].Target.String() < sorted[j].Target.String()
	})

	totalWeight := 0
	for _, backend := range sorted {
		if backend.Weight <= 0 {
			return nil, errors.New("Invalid backend weight 0")
		}
		totalWeight += backend.Weight
	}

	r := rand.Intn(totalWeight)
	pos := 0

	for _, backend := range sorted {
		pos += backend.Weight
		if r >= pos {
			continue
		}
		return backend, nil
	}

	return nil, errors.New("Cant elect backend")
}
