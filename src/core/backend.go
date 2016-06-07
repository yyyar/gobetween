/**
 * backend.go - backend definition
 *
 * @author Yaroslav Pogrebnyak <yyyaroslav@gmail.com>
 */

package core

import (
	"fmt"
	"math/big"
)

/**
 * Backend means upstream server
 * with all needed associate information
 */
type Backend struct {
	Target
	Priority int
	Weight   int
	Live     bool
	Stats    BackendStats
}

/**
 * Backend status
 */
type BackendStats struct {
	ActiveConnections int
	RxBytes           big.Int
	TxBytes           big.Int
}

/**
 * Check if backend equal to another
 */
func (this *Backend) EqualTo(other Backend) bool {
	return this.Target.EqualTo(other.Target)
}

/**
 * Merge another backend to this one
 */
func (this *Backend) MergeFrom(other Backend) *Backend {

	this.Priority = other.Priority
	this.Weight = other.Weight

	return this
}

/**
 * Get backends target address
 */
func (this *Backend) Address() string {
	return this.Target.Address()
}

/**
 * String conversion
 */
func (this Backend) String() string {
	return fmt.Sprintf("{%s p=%d,w=%d,l=%t,a=%d}",
		this.Address(), this.Priority, this.Weight, this.Live, this.Stats.ActiveConnections)
}
