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
	Priority int          `json:"priority"`
	Weight   int          `json:"weight"`
	Stats    BackendStats `json:"stats"`
}

/**
 * Backend status
 */
type BackendStats struct {
	Live              bool    `json:"live"`
	TotalConnections  int64   `json:"total_connections"`
	ActiveConnections int     `json:"active_connections"`
	RxBytes           big.Int `json:"rx"`
	TxBytes           big.Int `json:"tx"`
	RxSecond          big.Int `json:"rx_second"`
	TxSecond          big.Int `json:"tx_second"`
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
		this.Address(), this.Priority, this.Weight, this.Stats.Live, this.Stats.ActiveConnections)
}
