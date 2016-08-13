/**
 * stats.go - bandwidth stats
 *
 * @author Yaroslav Pogrebnyak <yyyaroslav@gmail.com>
 */

package counters

import (
	"../../core"
)

/**
 * Bandwidth stats object
 */
type BandwidthStats struct {

	// Total received bytes
	RxTotal uint64

	// Total transmitted bytes
	TxTotal uint64

	// Received bytes per second
	RxSecond uint

	// Transmitted bytes per second
	TxSecond uint

	// Optional target of stats
	Target core.Target
}
