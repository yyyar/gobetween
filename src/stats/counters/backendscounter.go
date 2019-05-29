package counters

/**
 * backendscounter.go - bandwidth counter for backends pool
 *
 * @author Yaroslav Pogrebnyak <yyyaroslav@gmail.com>
 */

import (
	"time"

	"github.com/yyyar/gobetween/core"
)

const (
	/* Stats update interval */
	INTERVAL = 2 * time.Second
)

/**
 * Bandwidth counter for backends pool
 */
type BackendsBandwidthCounter struct {

	/* Map of counters of specific targets */
	counters map[core.Target]*BandwidthCounter

	/* ----- channels ------ */

	/* Input channel of updated targets */
	In chan []core.Target

	/* Input channel of traffic deltas */
	Traffic chan core.ReadWriteCount

	/* Output channel for counted stats */
	Out chan BandwidthStats

	/* Stop channel */
	stop chan bool
}

/**
 * Creates new backends bandwidth counter
 */
func NewBackendsBandwidthCounter() *BackendsBandwidthCounter {
	return &BackendsBandwidthCounter{
		counters: make(map[core.Target]*BandwidthCounter),
		In:       make(chan []core.Target),
		Traffic:  make(chan core.ReadWriteCount),
		Out:      make(chan BandwidthStats),
		stop:     make(chan bool),
	}
}

/**
 * Start backends counter
 */
func (this *BackendsBandwidthCounter) Start() {

	go func() {
		for {
			select {

			// stop
			case <-this.stop:

				// Stop all counters
				for i := range this.counters {
					this.counters[i].Stop()
				}
				this.counters = nil

				// close channels
				close(this.In)
				close(this.Traffic)
				close(this.Out)
				return

			// new backends available
			case targets := <-this.In:
				this.UpdateCounters(targets)

			// new traffic available
			// route to appropriated counter
			case rwc := <-this.Traffic:
				counter, ok := this.counters[rwc.Target]
				// ignore stats for backend that is not is list
				if ok {
					counter.Traffic <- rwc
				}
			}

		}
	}()
}

/**
 * Update counters to match targets, optionally creating new
 * and deleting old counters
 */
func (this *BackendsBandwidthCounter) UpdateCounters(targets []core.Target) {

	result := map[core.Target]*BandwidthCounter{}

	// Keep or add needed workers
	for _, t := range targets {
		c, ok := this.counters[t]
		if !ok {
			c = NewBandwidthCounter(INTERVAL, this.Out)
			c.Target = t
			c.Start()
		}
		result[t] = c
	}

	// Stop needed counters
	for currentT, c := range this.counters {
		remove := true
		for _, t := range targets {
			if currentT.EqualTo(t) {
				remove = false
				break
			}
		}

		if remove {
			c.Stop()
		}
	}

	this.counters = result
}

/**
 * Stop backends counter
 */
func (this *BackendsBandwidthCounter) Stop() {
	this.stop <- true
}
