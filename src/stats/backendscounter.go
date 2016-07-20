/**
 * backendscounter.go - bandwidth counter for backends pool
 *
 * @author Yaroslav Pogrebnyak <yyyaroslav@gmail.com>
 */

package stats

import (
	"../core"
)

/**
 * Bandwidth counter for backends pool
 */
type BackendsBandwidthCounter struct {
	counters map[core.Target]*BandwidthCounter

	In        chan []core.Target
	InTraffic chan core.ReadWriteCount
	Out       chan BandwidthStats
	stop      chan bool
}

/**
 * Stop backends counter
 */
func (this *BackendsBandwidthCounter) Stop() {
	this.stop <- true
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
				close(this.In)
				close(this.InTraffic)
				close(this.Out)
				return

			// new backends available
			case targets := <-this.In:
				this.UpdateCounters(targets)

				// new traffic available
			case rwc := <-this.InTraffic:
				this.counters[rwc.Target].Traffic <- rwc
			}

		}
	}()
}

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

func NewBackendsBandwidthCounter() *BackendsBandwidthCounter {
	return &BackendsBandwidthCounter{
		counters:  make(map[core.Target]*BandwidthCounter),
		In:        make(chan []core.Target),
		InTraffic: make(chan core.ReadWriteCount),
		Out:       make(chan BandwidthStats),
		stop:      make(chan bool),
	}
}
