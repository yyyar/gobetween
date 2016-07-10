/**
 * handler.go - server stats handler
 *
 * @author Yaroslav Pogrebnyak <yyyaroslav@gmail.com>
 */

package stats

import (
	"../core"
	"math/big"
	"sync"
	"time"
)

const (
	/* Stats update interval */
	INTERVAL = 2 * time.Second
)

/**
 * Handlers Store
 */
var Store = struct {
	sync.RWMutex
	handlers map[string]*Handler
}{handlers: make(map[string]*Handler)}

/**
 * Get stats for the server
 */
func GetStats(name string) interface{} {

	Store.RLock()
	defer Store.RUnlock()

	handler, ok := Store.handlers[name]
	if !ok {
		return nil
	}
	return handler.stats // TODO: syncronize?
}

/**
 * Handler processess data from server
 */
type Handler struct {

	/* Server's name */
	name string

	/* Bandwidth counter */
	bandwidthCounter *BandwidthCounter

	/* Current stats */
	stats Stats

	/* ----- channels ----- */

	/* Server traffic data */
	Traffic chan core.ReadWriteCount

	/* Server current connections count */
	Connections chan int

	/* Current backends pool */
	Backends chan []core.Backend

	/* Channel for indicating stop request */
	stopChan chan bool
}

/**
 * Creates new stats handler for the server
 * with name 'name'
 */
func NewHandler(name string) *Handler {

	handler := &Handler{
		name:             name,
		bandwidthCounter: NewBandwidthCounter(INTERVAL),
		Traffic:          make(chan core.ReadWriteCount),
		Connections:      make(chan int),
		Backends:         make(chan []core.Backend),
		stopChan:         make(chan bool),
		stats: Stats{
			RxTotal:  big.NewInt(0),
			TxTotal:  big.NewInt(0),
			RxSecond: big.NewInt(0),
			TxSecond: big.NewInt(0),
			Backends: []core.Backend{},
		},
	}

	Store.Lock()
	Store.handlers[name] = handler
	Store.Unlock()

	return handler
}

/**
 * Request handler stop and clear resources
 */
func (this *Handler) Stop() {
	this.stopChan <- true
}

/**
 * Start handler work asynchroniously
 */
func (this *Handler) Start() {

	this.bandwidthCounter.Start()

	go func() {

		for {
			select {

			/* stop stats processor requested */
			case <-this.stopChan:
				this.bandwidthCounter.Stop()
				Store.Lock()
				delete(Store.handlers, this.name)
				Store.Unlock()
				close(this.Traffic)
				close(this.Connections)
				return

			case b := <-this.bandwidthCounter.Out:
				this.stats.RxTotal.Set(&b.RxTotal)
				this.stats.TxTotal.Set(&b.TxTotal)
				this.stats.RxSecond.Set(&b.RxSecond)
				this.stats.TxSecond.Set(&b.TxSecond)

			/* New traffic stats available */
			case rwc := <-this.Traffic:
				this.bandwidthCounter.Traffic <- rwc

			/* New backends available */
			case backends := <-this.Backends:
				this.stats.Backends = backends

			/* New connections count available */
			case connections := <-this.Connections:
				this.stats.ActiveConnections = connections
			}
		}
	}()

}
