package scheduler

/**
 * scheduler.go - schedule operations on backends and manages them
 *
 * @author Yaroslav Pogrebnyak <yyyaroslav@gmail.com>
 */

import (
	"fmt"
	"time"

	"github.com/yyyar/gobetween/core"
	"github.com/yyyar/gobetween/discovery"
	"github.com/yyyar/gobetween/healthcheck"
	"github.com/yyyar/gobetween/logging"
	"github.com/yyyar/gobetween/metrics"
	"github.com/yyyar/gobetween/stats"
	"github.com/yyyar/gobetween/stats/counters"
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
type ElectResponse struct {
	Backend *core.Backend
	Alive <-chan struct{}
}

type ElectRequest struct {
	Context  core.Context
	Response chan ElectResponse
	Err      chan error
}



/**
 * Scheduler
 */
type Scheduler struct {

	/* Balancer impl */
	Balancer core.Balancer

	/* Discovery impl */
	Discovery *discovery.Discovery

	/* Healthcheck impl */
	Healthcheck *healthcheck.Healthcheck

	CloseOnFailure bool

	/* ----- backends ------*/

	/* Current cached backends map */
	backends map[core.Target]*core.Backend

	liveness map[core.Target]chan struct{}

	/* Stats */
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

	log.Info("Starting scheduler ", this.StatsHandler.Name)

	this.ops = make(chan Op)
	this.elect = make(chan ElectRequest)
	this.stop = make(chan bool)
	this.backends = make(map[core.Target]*core.Backend)
	this.liveness = make(map[core.Target]chan struct{})

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
				log.Info("Stopping scheduler ", this.StatsHandler.Name)
				backendsPushTicker.Stop()
				this.Discovery.Stop()
				this.Healthcheck.Stop()
				metrics.RemoveServer(fmt.Sprintf("%s", this.StatsHandler.Name), this.backends)
				return
			}
		}
	}()
}

func (this *Scheduler) HandleBackendListChange() {
	this.Healthcheck.In <- this.Targets()
	this.StatsHandler.BackendsCounter.In <- this.Targets()
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

	metrics.ReportHandleBackendStatsChange(fmt.Sprintf("%s", this.StatsHandler.Name), target, this.backends)
}

/**
 * Updated backend live status
 */
func (this *Scheduler) HandleBackendLiveChange(target core.Target, live bool) {
	log := logging.For("scheduler")
	backend, ok := this.backends[target]
	if !ok {
		log.Warn("No backends for checkResult ", target)
		return
	}

	backend.Stats.Live = live

	terminationSignal, ok := this.liveness[target]
	if !live {
		if ok {
			log.Infof("Backend %s DOWN", backend.Target.String())
			close(terminationSignal)
			delete(this.liveness, target)
		}
	} else {
		if !ok {
			log.Infof("Backend %s UP", backend.Target.String())
			this.liveness[target] = make(chan struct{})
		}
	}

 	metrics.ReportHandleBackendLiveChange(fmt.Sprintf("%s", this.StatsHandler.Name), target, live)
}

/**
 * Update backends map
 */
func (this *Scheduler) HandleBackendsUpdate(backends []core.Backend) {
	changes := false

	// first mark all existing backends as not discovered
	for _, b := range this.backends {
		b.Stats.Discovered = false
	}

	for _, b := range backends {
		oldB, ok := this.backends[b.Target]

		if ok {
			// if we have this backend, update it's discovery properties
			oldB.MergeFrom(b)
			// mark found backend as discovered
			oldB.Stats.Discovered = true
			if _, ok := this.liveness[b.Target]; !ok && oldB.Stats.Live {
				this.liveness[b.Target] = make(chan struct{})
			}
			continue
		}

		b := b // b has to be local variable in order to make unique pointers
		b.Stats.Discovered = true
		this.backends[b.Target] = &b
		if b.Stats.Live {
			this.liveness[b.Target] = make(chan struct{})
		}
		changes = true
	}

	//remove not discovered backends without active connections
	for t, b := range this.backends {
		if b.Stats.Discovered {
			continue
		}
		if b.Stats.ActiveConnections > 0 {
			if this.CloseOnFailure {
				if l, ok := this.liveness[t]; ok {
					close(l)
					delete(this.liveness, t)
				}
			}
			continue
		}

		this.removeBackend(b)
		changes = true
	}
	if changes {
		this.HandleBackendListChange()
	}
}

func (this *Scheduler) removeBackend(b *core.Backend) {
	metrics.RemoveBackend(this.StatsHandler.Name, b)

	delete(this.backends, b.Target)
	delete(this.liveness, b.Target)
}

/**
 * Perform backend election
 */
func (this *Scheduler) HandleBackendElect(req ElectRequest) {

	// Filter only live and discovered backends
	var backends []*core.Backend
	for _, b := range this.backends {

		if !b.Stats.Live {
			continue
		}

		if !b.Stats.Discovered {
			continue
		}

		backends = append(backends, b)
	}

	// Elect backend
	backend, err := this.Balancer.Elect(req.Context, backends)
	if err != nil {
		req.Err <- err
		return
	}

	terminateSignal, ok := this.liveness[backend.Target]
	if !ok {
		// Should not happen, but if it does we can just create a new signal
		logging.For("scheduler").Warnf("Termination signal not found for backend %v", backend.Target)
		terminateSignal = make(chan struct{})
		this.liveness[backend.Target] = terminateSignal
	}

	req.Response <- ElectResponse{Backend: backend, Alive: terminateSignal}
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
		defer func() {
			if backend.Stats.ActiveConnections == 0 && !backend.Stats.Discovered {
				this.removeBackend(backend)
				this.HandleBackendListChange()
			}
		}()
	default:
		log.Warn("Don't know how to handle op ", op.op)
	}

	metrics.ReportHandleOp(fmt.Sprintf("%s", this.StatsHandler.Name), op.target, this.backends)
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
func (this *Scheduler) TakeBackend(context core.Context) (*core.Backend, <-chan struct{}, error) {
	r := ElectRequest{context, make(chan ElectResponse), make(chan error)}
	this.elect <- r
	select {
	case err := <-r.Err:
		return nil, nil, err
	case response := <-r.Response:
		return response.Backend, response.Alive, nil
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
