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
}

func (this *BackendsBandwidthCounter) Stop() {
}

func (this *BackendsBandwidthCounter) Start() {

	go func() {
		for {
			select {

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
			c = NewBandwidthCounter(INTERVAL)
			c.Target = t
			c.Start(this.Out)
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
	}
}
