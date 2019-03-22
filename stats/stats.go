package stats

/**
 * stats.go - server stats object
 *
 * @author Yaroslav Pogrebnyak <yyyaroslav@gmail.com>
 */

import (
	"github.com/yyyar/gobetween/core"
)

/**
 * Stats of the Server
 */
type Stats struct {

	/* Current active client connections */
	ActiveConnections uint `json:"active_connections"`

	/* Total received bytes from backend */
	RxTotal uint64 `json:"rx_total"`

	/* Total transmitter bytes to backend */
	TxTotal uint64 `json:"tx_total"`

	/* Received bytes to backend / second */
	RxSecond uint `json:"rx_second"`

	/* Transmitted bytes to backend / second */
	TxSecond uint `json:"tx_second"`

	/* Current backends pool */
	Backends []core.Backend `json:"backends"`
}
