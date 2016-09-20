/**
 * udp.go - udp session manager
 *
 * @author Illarion Kovalchuk <illarion.kovalchuk@gmail.com>
 * @author Yaroslav Pogrebnyak <yyyaroslav@gmail.com>
 */

package udp

import (
	"../../core"
	"../../logging"
	"../../stats"
	"../scheduler"
	"net"
	"time"
)

type request struct {
	addr     string
	response chan *session
}

/**
 * SessionManager emulates UDP "session" and manages them
 */
type sessionManager struct {
	sessions     map[string]*session
	statsHandler *stats.Handler
	sessionCount uint
	addC         chan *session
	remC         chan *session
	stopC        chan bool
	getC         chan *request
}

/**
 * Creates new session manager
 */
func newSessionManager(statsHandler *stats.Handler) *sessionManager {
	return &sessionManager{
		statsHandler: statsHandler,
		sessions:     make(map[string]*session),
		sessionCount: 0,
		addC:         make(chan *session),
		remC:         make(chan *session),
		stopC:        make(chan bool),
		getC:         make(chan *request),
	}
}

/**
 * Creates new sessions; adds to itself and returns it
 */
func (sm *sessionManager) createSession(addr *net.UDPAddr, scheduler *scheduler.Scheduler, backend *core.Backend) *session {

	log := logging.For("udp.SessionManager.createSession")

	backendAddr, err := net.ResolveUDPAddr("udp", backend.Target.String())
	if err != nil {
		log.Error("Error ResolveUDPAddr: ", err)
		return nil
	}

	backendConn, err := net.DialUDP("udp", nil, backendAddr)

	if err != nil {
		log.Debug("Error connecting to backend: ", err)
		return nil
	}

	scheduler.IncrementConnection(*backend)

	session := &session{
		clientAddr:   addr,
		statsHandler: sm.statsHandler,
		scheduler:    scheduler,
		backend:      backend,
		backendConn:  backendConn,
		lastUpdated:  time.Now(),
		touchC:       make(chan bool),
		stopC:        make(chan bool),
	}

	sm.add(session)
	return session
}

/**
 * Starts session manager processing
 */
func (sm *sessionManager) start() {
	go func() {
		for {
			select {

			/* Handle adding new session */
			case session := <-sm.addC:
				sm.sessionCount++
				sm.statsHandler.Connections <- sm.sessionCount
				sm.sessions[session.clientAddr.String()] = session

			/* Handle removig expired session */
			case session := <-sm.remC:
				sm.sessionCount--
				sm.statsHandler.Connections <- sm.sessionCount
				delete(sm.sessions, session.clientAddr.String())

			/* Handle get session request */
			case request := <-sm.getC:
				session, ok := sm.sessions[request.addr]
				if ok {
					request.response <- session
				} else {
					request.response <- nil
				}

			/* Handle stop session manager */
			case <-sm.stopC:
				for _, session := range sm.sessions {
					session.stop()
				}
			}

		}
	}()
}

/**
 * Returns sesion for client if exists
 */
func (sm *sessionManager) getForAddr(clientAddr *net.UDPAddr) (*session, bool) {
	request := &request{
		addr:     clientAddr.String(),
		response: make(chan *session),
	}
	sm.getC <- request
	session := <-request.response
	return session, session != nil
}

/**
 * Adds session
 */
func (sm *sessionManager) add(session *session) {
	go func() {
		sm.addC <- session
	}()
}

/**
 * Removes session
 */
func (sm *sessionManager) remove(session *session) {
	go func() {
		sm.remC <- session
	}()
}

/**
 * Stops session manager
 */
func (sm *sessionManager) stop() {
	go func() {
		sm.stopC <- true
	}()
}
