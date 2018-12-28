/**
 * iphash.go - iphash2 balance implementation
 *
 * @author Yaroslav Pogrebnyak <yyyaroslav@gmail.com>
 */

package balance

import (
	"errors"
	"github.com/yyyar/gobetween/config"
	"github.com/yyyar/gobetween/core"
	"github.com/yyyar/gobetween/logging"
	"sync"
	"time"
)

/**
 * Iphash2 balancer impelemts "sticky" iphash load balancing.
 */
type Iphash2Balancer struct {

	/* configuration */
	cfg config.IpHash2BalanceConfig

	duration time.Duration

	/* sticky table mapping */
	/* ip str -> session */
	table map[string]*Session

	mutex sync.Mutex
}

/**
 * Iphash balancing session
 */
type Session struct {
	timer   *time.Timer
	backend *core.Backend
	mutex   sync.Mutex
	done    chan bool
	keep    bool
}

/**
 * Constructor
 */
func NewIphash2Balancer(cfg config.BalanceConfig) interface{} {
	b := &Iphash2Balancer{
		cfg:   *cfg.IpHash2BalanceConfig,
		table: map[string]*Session{},
		mutex: sync.Mutex{},
	}

	b.duration, _ = time.ParseDuration(cfg.IpHash2Expire)

	return b
}

/**
 * Elect backend using iphash strategy
 * This balancer is stable in both adding and removing backends
 * It keeps mapping cache for some period of time.
 *
 * TODO: probably not the best implementation because of extensive usage
 *       of complex locks. Consider re-doing keeping high performance,
 *       probably using jobber goroutine for removing expired sessions.
 */
func (b *Iphash2Balancer) Elect(context core.Context, backends []*core.Backend) (*core.Backend, error) {

	log := logging.For("balance/iphash2")

	if len(backends) == 0 {
		return nil, errors.New("Can't elect backend, Backends empty")
	}

	log.Info(b.duration)
	// --------------- lock balancer

	b.mutex.Lock()

	sess, ok := b.table[context.Ip().String()]
	if !ok {
		backend, err := ((*WeightBalancer)(nil)).Elect(context, backends)

		b.table[context.Ip().String()] = &Session{
			backend: backend,
			mutex:   sync.Mutex{},
			done:    make(chan bool, 1),
			keep:    false,
			timer: time.AfterFunc(b.duration, func() {
				log := logging.For("balance/iphash2")
				log.Debug("Begin cleanup session ", context.Ip().String())
				b.mutex.Lock()
				sess = b.table[context.Ip().String()]
				if !sess.keep {
					delete(b.table, context.Ip().String())
				}
				b.mutex.Unlock()
				sess.done <- true
				log.Debug("End cleanup session ", context.Ip().String())
			}),
		}

		b.mutex.Unlock()
		return backend, err
	}

	/* now lock on the session to allow other clients work well in elect */

	sess.mutex.Lock()
	defer sess.mutex.Unlock()

	stopped := sess.timer.Stop()
	if !stopped {

		log.Info("in !stopped")
		// --------------- unlock balancer
		sess.keep = true // it's in sync guaranted by b.mutex lock
		b.mutex.Unlock()

		// wait cleanup goroutine to finish to ensure it will
		// not cleanup session after we reset it here
		<-sess.done

		// put sess back to table since it was removed in AfterFunc
		// and we sure it already completed here
		// TODO: ENSURE OTHER CLIENTS DO NOT START CREATING NEW SESSION WHILE WE DO NOT LOCKED
		b.mutex.Lock()
		sess.keep = false
		b.table[context.Ip().String()] = sess
		b.mutex.Unlock()
	}

	b.mutex.Unlock()

	sess.timer.Reset(b.duration)

	// check if previously elected backends still presents in backends list
	for _, backend := range backends {
		if backend.Address() == sess.backend.Address() {
			return sess.backend, nil
		}
	}

	// previously elected backend died, elect new one
	backend, err := ((*WeightBalancer)(nil)).Elect(context, backends)
	sess.backend = backend

	return backend, err
}
