/**
 * handler.go - server stats handler
 *
 * @author Yaroslav Pogrebnyak <yyyaroslav@gmail.com>
 */

package stats

import (
	"../core"
	"math/big"
	"time"
)

const (
	/* Stats update interval */
	INTERVAL = 2 * time.Second
)

// TODO: Add sync here and to other places in thgis file
var Store = make(map[string]*Handler)

/**
 * Get stats for the server
 * TODO: Sync it!
 */
func GetStats(name string) interface{} {
	handler, ok := Store[name]
	if !ok {
		return nil
	}
	return handler.stats
}

/**
 * Handler processess data from server
 */
type Handler struct {

	/* Server's name */
	name string

	/* Ticker for periodic pushing stats results */
	ticker *time.Ticker

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
		name:        name,
		ticker:      time.NewTicker(INTERVAL),
		Traffic:     make(chan core.ReadWriteCount),
		Connections: make(chan int),
		Backends:    make(chan []core.Backend),
		stopChan:    make(chan bool),
		stats: Stats{
			RxTotal:  big.NewInt(0),
			TxTotal:  big.NewInt(0),
			RxSecond: big.NewInt(0),
			TxSecond: big.NewInt(0),
			Backends: []core.Backend{},
		},
	}

	Store[name] = handler
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

	go func() {

		//collector := NewCollector(this.name)

		lastRxTotal := big.NewInt(0)
		lastTxTotal := big.NewInt(0)

		newTxRx := false
		for {
			select {

			/* stop stats processor requested */
			case <-this.stopChan:
				this.ticker.Stop()
				// remove from store
				delete(Store, this.name)
				close(this.Traffic)
				close(this.Connections)
				return

			/* prepare and push next stats update */
			case <-this.ticker.C:

				if !newTxRx {
					this.stats.RxSecond = big.NewInt(0)
					this.stats.TxSecond = big.NewInt(0)
				} else {

					dRx := big.NewInt(0).Sub(this.stats.RxTotal, lastRxTotal)
					dTx := big.NewInt(0).Sub(this.stats.TxTotal, lastTxTotal)

					this.stats.RxSecond.Div(dRx, big.NewInt(int64(INTERVAL.Seconds())))
					this.stats.TxSecond.Div(dTx, big.NewInt(int64(INTERVAL.Seconds())))

					lastRxTotal.Set(this.stats.RxTotal)
					lastTxTotal.Set(this.stats.TxTotal)

					newTxRx = false
				}

			/* New traffic stats available */
			case rwc := <-this.Traffic:
				newTxRx = true
				this.stats.RxTotal.Add(this.stats.RxTotal, big.NewInt(int64(rwc.CountRead)))
				this.stats.TxTotal.Add(this.stats.TxTotal, big.NewInt(int64(rwc.CountWrite)))

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
