package counters

/**
 * counter.go - bandwidth counter
 *
 * @author Yaroslav Pogrebnyak <yyyaroslav@gmail.com>
 */

import (
	"time"

	"github.com/yyyar/gobetween/core"
)

/**
 * Count total bandwidth and bandwidth per second
 */
type BandwidthCounter struct {

	/* Bandwidth Stats */
	BandwidthStats

	/* Last received total bytes */
	RxTotalLast uint64
	/* Last transmitted total bytes */
	TxTotalLast uint64

	/* Timeframe to calculate per-second bandwidth */
	interval time.Duration
	/* Ticker for per-second bandwidth calculation and pushing stats */
	ticker *time.Ticker

	/* Indicates that new bandwidth delta was received */
	newTxRx bool

	/* ----- channels ----- */

	/* Input channel for bandwidth deltas */
	Traffic chan core.ReadWriteCount

	/* Stop channel */
	stop chan bool

	/* Output channel for bandwidth stats */
	Out chan BandwidthStats
}

/**
 * Create new BandwidthCounter
 */
func NewBandwidthCounter(interval time.Duration, out chan BandwidthStats) *BandwidthCounter {

	return &BandwidthCounter{
		interval: interval,
		ticker:   time.NewTicker(interval),
		BandwidthStats: BandwidthStats{
			RxTotal: 0,
			TxTotal: 0,
		},
		TxTotalLast: 0,
		RxTotalLast: 0,
		Out:         out,
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
				return

				// New counting cycle
			case <-this.ticker.C:

				if !this.newTxRx {
					this.RxSecond = 0
					this.TxSecond = 0
				} else {

					dRx := this.RxTotal - this.RxTotalLast
					dTx := this.TxTotal - this.TxTotalLast

					this.RxSecond = uint(dRx / uint64(this.interval.Seconds()))
					this.TxSecond = uint(dTx / uint64(this.interval.Seconds()))

					this.RxTotalLast = this.RxTotal
					this.TxTotalLast = this.TxTotal

					this.newTxRx = false
				}

				// Send results to out
				this.Out <- this.BandwidthStats

				// New traffic deltas available
			case rwc := <-this.Traffic:
				this.newTxRx = true
				this.RxTotal += uint64(rwc.CountRead)
				this.TxTotal += uint64(rwc.CountWrite)
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
