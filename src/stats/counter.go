/**
 * counter.go - bandwidth counter
 *
 * @author Yaroslav Pogrebnyak <yyyaroslav@gmail.com>
 */

package stats

import (
	"../core"
	"math/big"
	"time"
)

/**
 * Bandwidth stats object
 */
type BandwidthStats struct {

	// Total received bytes
	RxTotal big.Int

	// Total transmitted bytes
	TxTotal big.Int

	// Received bytes per second
	RxSecond big.Int

	// Transmitted bytes per second
	TxSecond big.Int
}

/**
 * Count total bandwidth and bandwidth per second
 */
type BandwidthCounter struct {
	perSecondRx *big.Int
	perSecondTx *big.Int

	totalRx *big.Int
	totalTx *big.Int

	lastRx *big.Int
	lastTx *big.Int

	interval time.Duration // Per-second counter timeframe
	ticker   *time.Ticker

	newTxRx bool // Indicates that new bandwidth delta received

	/* ----- channels ----- */

	// Output channel for bandwidth stats
	Out chan BandwidthStats

	// Input channel for bandwidth deltas
	Traffic chan core.ReadWriteCount

	// Stop indicator
	stop chan bool
}

/**
 * Create new BandwidthCounter
 */
func NewBandwidthCounter(interval time.Duration) *BandwidthCounter {

	return &BandwidthCounter{
		interval:    interval,
		ticker:      time.NewTicker(interval),
		perSecondRx: big.NewInt(0),
		perSecondTx: big.NewInt(0),
		lastRx:      big.NewInt(0),
		lastTx:      big.NewInt(0),
		totalRx:     big.NewInt(0),
		totalTx:     big.NewInt(0),
		Out:         make(chan BandwidthStats),
		Traffic:     make(chan core.ReadWriteCount),
		stop:        make(chan bool),
	}
}

/**
 * Starts bandwidth counter
 */
func (this *BandwidthCounter) Start() {

	go func() {
		for {
			select {

			// Stop requested
			case <-this.stop:
				this.ticker.Stop()
				close(this.Traffic)
				close(this.Out)
				return

			// New counting cycle
			case <-this.ticker.C:

				if !this.newTxRx {
					this.perSecondRx = big.NewInt(0)
					this.perSecondTx = big.NewInt(0)
				} else {

					dRx := big.NewInt(0).Sub(this.totalRx, this.lastRx)
					dTx := big.NewInt(0).Sub(this.totalTx, this.lastTx)

					this.perSecondRx.Div(dRx, big.NewInt(int64(this.interval.Seconds())))
					this.perSecondTx.Div(dTx, big.NewInt(int64(this.interval.Seconds())))

					this.lastRx.Set(this.totalRx)
					this.lastTx.Set(this.totalTx)

					this.newTxRx = false
				}

				// Send results to out
				this.Out <- BandwidthStats{
					RxTotal:  *this.totalRx,
					TxTotal:  *this.totalTx,
					RxSecond: *this.perSecondRx,
					TxSecond: *this.perSecondTx,
				}

			// New traffic deltas available
			case rwc := <-this.Traffic:
				this.newTxRx = true
				this.totalRx.Add(this.totalRx, big.NewInt(int64(rwc.CountRead)))
				this.totalTx.Add(this.totalTx, big.NewInt(int64(rwc.CountWrite)))
			}
		}
	}()
}

/**
 * Stops bandwidth counter
 */
func (this *BandwidthCounter) Stop() {
	this.stop <- true
}
