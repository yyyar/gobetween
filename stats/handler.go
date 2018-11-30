package stats

/**
 * handler.go - server stats handler
 *
 * @author Yaroslav Pogrebnyak <yyyaroslav@gmail.com>
 */

import (
	"fmt"
	"time"

	"github.com/yyyar/gobetween/core"
	"github.com/yyyar/gobetween/metrics"
	"github.com/yyyar/gobetween/stats/counters"
)

const (
	/* Stats update interval */
	INTERVAL = 2 * time.Second
)

/**
 * Handler processess data from server
 */
type Handler struct {

	/* Server's name */
	Name string

	/* Server counter */
	serverCounter *counters.BandwidthCounter
	/* Backends counters */
	BackendsCounter *counters.BackendsBandwidthCounter

	/* Current stats */
	latestStats Stats

	/* ----- channels ----- */

	/* Server traffic data */
	Traffic chan core.ReadWriteCount

	/* Server current connections count */
	Connections chan uint

	/* Current backends pool */
	Backends chan []core.Backend

	/* Channel for indicating stop request */
	stopChan chan bool

	/* Input channel for latest stats */
	ServerStats chan counters.BandwidthStats
}

/**
 * Creates new stats handler for the server
 * with name 'name'
 */
func NewHandler(name string) *Handler {

	handler := &Handler{
		Name:        name,
		ServerStats: make(chan counters.BandwidthStats, 1),
		Traffic:     make(chan core.ReadWriteCount),
		Connections: make(chan uint),
		Backends:    make(chan []core.Backend),
		stopChan:    make(chan bool),
		latestStats: Stats{
			RxTotal:  0,
			TxTotal:  0,
			RxSecond: 0,
			TxSecond: 0,
			Backends: []core.Backend{},
		},
	}

	handler.serverCounter = counters.NewBandwidthCounter(INTERVAL, handler.ServerStats)
	handler.BackendsCounter = counters.NewBackendsBandwidthCounter()

	Store.Lock()
	Store.handlers[name] = handler
	Store.Unlock()

	return handler
}

/**
 * Start handler work asynchroniously
 */
func (this *Handler) Start() {

	this.serverCounter.Start()
	this.BackendsCounter.Start()

	go func() {

		for {
			select {

			/* stop stats processor requested */
			case <-this.stopChan:

				this.serverCounter.Stop()
				this.BackendsCounter.Stop()

				Store.Lock()
				delete(Store.handlers, this.Name)
				Store.Unlock()

				// close channels
				close(this.ServerStats)
				close(this.Traffic)
				close(this.Connections)
				return

			/* New server stats available */
			case b := <-this.ServerStats:
				this.latestStats.RxTotal = b.RxTotal
				this.latestStats.TxTotal = b.TxTotal
				this.latestStats.RxSecond = b.RxSecond
				this.latestStats.TxSecond = b.TxSecond

				metrics.ReportHandleStatsChange(fmt.Sprintf("%s", this.Name), b)

			/* New server backends with stats available */
			case backends := <-this.Backends:
				this.latestStats.Backends = backends

			/* New sever connections count available */
			case connections := <-this.Connections:
				this.latestStats.ActiveConnections = connections

				metrics.ReportHandleConnectionsChange(fmt.Sprintf("%s", this.Name), connections)

			/* New traffic stats available */
			case rwc := <-this.Traffic:
				// forward to counters
				go func() {
					this.serverCounter.Traffic <- rwc
					this.BackendsCounter.Traffic <- rwc
				}()
			}
		}
	}()

}

/**
 * Request handler stop and clear resources
 */
func (this *Handler) Stop() {
	this.stopChan <- true
}
