/**
 * scheduler.go - schedule operations on backends and manages them
 *
 * @author Yaroslav Pogrebnyak <yyyaroslav@gmail.com>
 */

package server

import (
	"../balance"
	"../core"
	"../discovery"
	"../healthcheck"
	"../logging"
	"../stats"
	"../stats/counters"
	"time"
)

/**
 * Backend Operation action
 */
type OpAction int

/**
 * Constants for backend operation
 */
const (
	IncrementConnection OpAction = iota
	DecrementConnection
	IncrementRefused
	IncrementTx
	IncrementRx
)

/**
 * Operation on backend
 */
type Op struct {
	target core.Target
	op     OpAction
	param  interface{}
}

/**
 * Request to elect backend
 */
type ElectRequest struct {
	Context  core.Context
	Response chan core.Backend
	Err      chan error
}

/**
 * Scheduler
 */
type Scheduler struct {

	/* Balancer impl */
	Balancer balance.Balancer

	/* Discovery impl */
	Discovery *discovery.Discovery

	/* Healthcheck impl */
	Healthcheck *healthcheck.Healthcheck

	/* Current cached backends map */
	backends map[core.Target]*core.Backend

	StatsHandler *stats.Handler

	/* ----- channels ----- */

	/* Backend operation channel */
	ops chan Op

	/* Stop channel */
	stop chan bool

	/* Elect backend channel */
	elect chan ElectRequest
}

/**
 * Start scheduler
 */
func (this *Scheduler) Start() {

	log := logging.For("scheduler")

	log.Info("Starting scheduler")

	this.ops = make(chan Op)
	this.elect = make(chan ElectRequest)
	this.stop = make(chan bool)

	this.Discovery.Start()
	this.Healthcheck.Start()

	// backends stats pusher ticker
	backendsPushTicker := time.NewTicker(2 * time.Second)

	/**
	 * Goroutine updates and manages backends
	 */
	go func() {
		for {
			select {

			/* ----- discovery ----- */

			// handle newly discovered backends
			case backends := <-this.Discovery.Discover():
				this.HandleBackendsUpdate(backends)
				this.Healthcheck.In <- this.Targets()
				this.StatsHandler.BackendsCounter.In <- this.Targets()

			/* ------ healthcheck ----- */

			// handle backend healthcheck result
			case checkResult := <-this.Healthcheck.Out:
				this.HandleBackendLiveChange(checkResult.Target, checkResult.Live)

			/* ----- stats ----- */

			// push current backends to stats handler
			case <-backendsPushTicker.C:
				this.StatsHandler.Backends <- this.Backends()

			// handle new bandwidth stats of a backend
			case bs := <-this.StatsHandler.BackendsCounter.Out:
				this.HandleBackendStatsChange(bs.Target, &bs)

			/* ----- operations ----- */

			// handle backend operation
			case op := <-this.ops:
				this.HandleOp(op)

			// elect backend
			case electReq := <-this.elect:
				this.HandleBackendElect(electReq)

			/* ----- stop ----- */

			// handle scheduler stop
			case <-this.stop:
				log.Info("Stopping scheduler")
				backendsPushTicker.Stop()
				this.Discovery.Stop()
				this.Healthcheck.Stop()
				return
			}
		}
	}()
}

/**
 * Returns targets of current backends
 */
func (this *Scheduler) Targets() []core.Target {

	keys := make([]core.Target, 0, len(this.backends))
	for k := range this.backends {
		keys = append(keys, k)
	}

	return keys
}

/**
 * Return current backends
 */
func (this *Scheduler) Backends() []core.Backend {

	backends := make([]core.Backend, 0, len(this.backends))
	for _, b := range this.backends {
		backends = append(backends, *b)
	}

	return backends
}

/**
 * Updated backend stats
 */
func (this *Scheduler) HandleBackendStatsChange(target core.Target, bs *counters.BandwidthStats) {

	backend, ok := this.backends[target]
	if !ok {
		logging.For("scheduler").Warn("No backends for checkResult ", target)
		return
	}

	backend.Stats.RxBytes = bs.RxTotal
	backend.Stats.TxBytes = bs.TxTotal
	backend.Stats.RxSecond = bs.RxSecond
	backend.Stats.TxSecond = bs.TxSecond
}

/**
 * Updated backend live status
 */
func (this *Scheduler) HandleBackendLiveChange(target core.Target, live bool) {

	backend, ok := this.backends[target]
	if !ok {
		logging.For("scheduler").Warn("No backends for checkResult ", target)
		return
	}

	backend.Stats.Live = live
}

/**
 * Update backends map
 */
func (this *Scheduler) HandleBackendsUpdate(backends []core.Backend) {

	updated := map[core.Target]*core.Backend{}

	for i := range backends {
		b := backends[i]
		oldB, ok := this.backends[b.Target]

		if ok {
			// if we have this backend, update it's discovery properties
			updated[oldB.Target] = oldB.MergeFrom(b)
		} else {
			updated[b.Target] = &b
		}
	}

	this.backends = updated
}

/**
 * Perform backend election
 */
func (this *Scheduler) HandleBackendElect(req ElectRequest) {

	// Filter only live backends
	var backends []core.Backend
	for _, b := range this.backends {
		if b.Stats.Live {
			backends = append(backends, *b)
		}
	}

	// Elect backend
	backend, err := this.Balancer.Elect(&req.Context, backends)
	if err != nil {
		req.Err <- err
		return
	}

	req.Response <- *backend
}

/**
 * Handle operation on the backend
 */
func (this *Scheduler) HandleOp(op Op) {

	// Increment global counter, even if
	// backend for this count may be out of discovery pool
	switch op.op {
	case IncrementTx:
		this.StatsHandler.Traffic <- core.ReadWriteCount{CountWrite: op.param.(uint), Target: op.target}
		return
	case IncrementRx:
		this.StatsHandler.Traffic <- core.ReadWriteCount{CountRead: op.param.(uint), Target: op.target}
		return
	}

	log := logging.For("scheduler")

	backend, ok := this.backends[op.target]
	if !ok {
		log.Warn("Trying op ", op.op, " on not tracked target ", op.target)
		return
	}

	switch op.op {
	case IncrementRefused:
		backend.Stats.RefusedConnections++
	case IncrementConnection:
		backend.Stats.ActiveConnections++
		backend.Stats.TotalConnections++
	case DecrementConnection:
		backend.Stats.ActiveConnections--
	default:
		log.Warn("Don't know how to handle op ", op.op)
	}

}

/**
 * Stop scheduler
 */
func (this *Scheduler) Stop() {
	this.stop <- true
}

/**
 * Take elect backend for proxying
 */
func (this *Scheduler) TakeBackend(context core.Context) (*core.Backend, error) {
	r := ElectRequest{context, make(chan core.Backend), make(chan error)}
	this.elect <- r
	select {
	case err := <-r.Err:
		return nil, err
	case backend := <-r.Response:
		return &backend, nil
	}
}

/**
 * Increment connection refused count for backend
 */
func (this *Scheduler) IncrementRefused(backend core.Backend) {
	this.ops <- Op{backend.Target, IncrementRefused, nil}
}

/**
 * Increment backend connection counter
 */
func (this *Scheduler) IncrementConnection(backend core.Backend) {
	this.ops <- Op{backend.Target, IncrementConnection, nil}
}

/**
 * Decrement backends connection counter
 */
func (this *Scheduler) DecrementConnection(backend core.Backend) {
	this.ops <- Op{backend.Target, DecrementConnection, nil}
}

/**
 * Increment Rx stats for backend
 */
func (this *Scheduler) IncrementRx(backend core.Backend, c uint) {
	this.ops <- Op{backend.Target, IncrementRx, c}
}

/**
 * Increment Tx stats for backends
 */
func (this *Scheduler) IncrementTx(backend core.Backend, c uint) {
	this.ops <- Op{backend.Target, IncrementTx, c}
}
