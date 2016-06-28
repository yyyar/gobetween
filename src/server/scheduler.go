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
)

/**
 * Operation on backend
 */
type Op struct {
	target core.Target
	op     OpAction
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
	balancer balance.Balancer

	/* Discovery impl */
	discovery *discovery.Discovery

	/* Healthcheck impl */
	healthcheck *healthcheck.Healthcheck

	/* Current cached backends map */
	backends map[core.Target]*core.Backend

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
func (this *Scheduler) start() {

	log := logging.For("scheduler")

	log.Info("Starting scheduler")

	this.ops = make(chan Op)
	this.elect = make(chan ElectRequest)
	this.stop = make(chan bool)

	this.discovery.Start()
	this.healthcheck.Start()

	/**
	 * Goroutine updates and manages backends
	 */
	go func() {
		for {
			select {

			// handle newly discovered backends
			case backends := <-this.discovery.Discover():
				this.HandleBackendsUpdate(backends)
				this.healthcheck.In <- this.Targets()

			// handle backend healthcheck result
			case checkResult := <-this.healthcheck.Out:
				this.HandleBackendLiveChange(checkResult.Target, checkResult.Live)

			// handle backend operation
			case op := <-this.ops:
				this.HandleOp(op)

			// elect backend
			case electReq := <-this.elect:
				this.HandleBackendElect(electReq)

			// handle scheduler stop
			case <-this.stop:
				log.Info("Stopping scheduler")
				this.discovery.Stop()
				this.healthcheck.Stop()
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
 * Updated backend live status
 */
func (this *Scheduler) HandleBackendLiveChange(target core.Target, live bool) {

	log := logging.For("scheduler")

	backend, ok := this.backends[target]
	if !ok {
		log.Warn("No backends for checkResult ", target)
		return
	}

	backend.Live = live
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
		if b.Live {
			backends = append(backends, *b)
		}
	}

	// Elect backend
	backend, err := this.balancer.Elect(&req.Context, backends)
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

	log := logging.For("scheduler")

	backend, ok := this.backends[op.target]
	if !ok {
		log.Warn("Trying op ", op.op, " on not tracked target ", op.target)
		return
	}

	switch op.op {
	case IncrementConnection:
		backend.Stats.ActiveConnections++
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
func (this *Scheduler) TakeBackend(context *core.Context) (*core.Backend, error) {
	r := ElectRequest{*context, make(chan core.Backend), make(chan error)}
	this.elect <- r
	select {
	case err := <-r.Err:
		return nil, err
	case backend := <-r.Response:
		return &backend, nil
	}
}

/**
 * Increment backend connection counter
 */
func (this *Scheduler) IncrementConnection(backend core.Backend) {
	this.ops <- Op{backend.Target, IncrementConnection}
}

/**
 * Decrement backends connection counter
 */
func (this *Scheduler) DecrementConnection(backend core.Backend) {
	this.ops <- Op{backend.Target, DecrementConnection}
}
