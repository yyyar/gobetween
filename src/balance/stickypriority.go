/**
 * StickyPriority.go - priority based "sticky session" balance implementation
 * allow for a "preferred backend" for new sessions, whilst keeping old session on existing backends
 *
 * @author quedunk <quedunk@gmail.com>
 */

package balance

import (
	"../config"
	"../core"
	"../logging"
	"errors"
	"time"
)

/**
 * balancer implements "sticky" priority based balancing.
 */
type StickyPriorityBalancer struct {
	duration time.Duration

	/* sticky table mapping */
	/* ip str -> session */
	table map[string]*StickyPrioritySession
}

/**
 * StickyPriority balancing session
 */
type StickyPrioritySession struct {
	backend		*core.Backend
	timer		*time.Timer
	lasttouch	time.Time
	needstimer	bool
}

/**
 * Constructor
 */
func NewStickyPriorityBalancer(cfg config.BalanceConfig) interface{} {

	b := &StickyPriorityBalancer{
		table: map[string]*StickyPrioritySession{},
	}

	b.duration, _ = time.ParseDuration(cfg.StickyPrioritySessionExpire)

	return b
}

/**
 * Elect backend using priority strategy
 * It keeps mapping cache for some period of time.
 */
func (b *StickyPriorityBalancer) Elect(context core.Context, backends []*core.Backend) (*core.Backend, error) {
	log := logging.For("balance/StickyPriority")

	if len(backends) == 0 {
		return nil, errors.New("Can't elect backend, Backends empty")
	}

	log.Debug("Looking for ", context.Ip())
	var backend *core.Backend
	var err error
	sess, ok := b.table[context.Ip().String()]
	if !ok {
		// if we couldnt find an existing session, make one + give it a valid backend
		backend, err = ((*PriorityBalancer)(nil)).Elect(context, backends)
		b.table[context.Ip().String()] = &StickyPrioritySession{
			backend:	backend,
		}
		sess = b.table[context.Ip().String()]
		sess.needstimer = true
		log.Debug("client " , context.Ip() , " new session on backend ", sess.backend.Address())

	} else {
		// got a session, check if previously elected backend is valid
		for _, validbackend := range backends {
			if validbackend.Address() == sess.backend.Address() {
				log.Debug("client ", context.Ip(), " found existing valid backend ", sess.backend.Address())
				backend = validbackend
				break
			}
		}
		// couldnt find the old backend? get a new one!
		if (backend == nil) {
			backend, err = ((*PriorityBalancer)(nil)).Elect(context, backends)
			log.Debug("client ", context.Ip(), " existing backend not valid, selected new one ", sess.backend.Address())
			sess.backend = backend
		}
	}

	// update session expiry time + set up a timer to clean up once expiry time has been reached
	sess.lasttouch = time.Now()
        if (sess.needstimer) {
		sess.needstimer = false;
		setTimer(context, *b)
	}

	return backend, err
}

func setTimer(context core.Context, b StickyPriorityBalancer) {
	log := logging.For("balance/StickyPriority/setTimer")

	log.Debug("client " , context.Ip().String(), " setting expiry check")

	sess := b.table[context.Ip().String()]
	// expiry seconds is; lasttouch + duration of expiry - timenow. 
	expirysecs := sess.lasttouch.Add(b.duration).Sub(time.Now())

	// if expirysecs < 0, then afterfunc will ignore it (accoring to sleep.go doco)
	sess.timer = time.AfterFunc(expirysecs, func() {
		// wait for the timer to expiry, then do this to see if we need to clean up:
		log.Debug("client " , context.Ip().String(), " timer triggered")
		sess := b.table[context.Ip().String()]
		if (sess != nil) {
			log.Debug("client " , context.Ip().String(), " found existing session")
			if (time.Now().After(sess.lasttouch.Add(b.duration))) {
				log.Debug("client " , context.Ip().String(), " session expired")
				delete(b.table, context.Ip().String())
				log.Debug("client " , context.Ip().String(), " session deleted")
			} else {
				log.Debug("client " , context.Ip().String(), " session not expired, setting new timer")
				setTimer(context, b)
			}
		}
	})
}
