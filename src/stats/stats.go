/**
 * stats.go - server stats object
 *
 * @author Yaroslav Pogrebnyak <yyyaroslav@gmail.com>
 */

package stats

import (
	"../core"
	"math/big"
)

/**
 * Stats of the Server
 */
type Stats struct {

	/* Current active client connections */
	ActiveConnections int `json:"active_connections"`

	/* Total received bytes from backend */
	RxTotal *big.Int `json:"rx_total"`

	/* Total transmitter bytes to backend */
	TxTotal *big.Int `json:"tx_total"`

	/* Received bytes to backend / second */
	RxSecond *big.Int `json:"rx_second"`

	/* Transmitted bytes to backend / second */
	TxSecond *big.Int `json:"tx_second"`

	/* Current backends pool */
	Backends []core.Backend `json:"backends"`
}
