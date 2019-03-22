package core

/**
 * backend.go - backend definition
 *
 * @author Yaroslav Pogrebnyak <yyyaroslav@gmail.com>
 */

import (
	"fmt"
)

/**
 * Backend means upstream server
 * with all needed associate information
 */
type Backend struct {
	Target
	Priority int          `json:"priority"`
	Weight   int          `json:"weight"`
	Sni      string       `json:"sni,omitempty"`
	Stats    BackendStats `json:"stats"`
}

/**
 * Backend status
 */
type BackendStats struct {
	Live               bool   `json:"live"`
	Discovered         bool   `json:"discovered"`
	TotalConnections   int64  `json:"total_connections"`
	ActiveConnections  uint   `json:"active_connections"`
	RefusedConnections uint64 `json:"refused_connections"`
	RxBytes            uint64 `json:"rx"`
	TxBytes            uint64 `json:"tx"`
	RxSecond           uint   `json:"rx_second"`
	TxSecond           uint   `json:"tx_second"`
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
	this.Sni = other.Sni

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
