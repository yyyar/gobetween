/**
 * manager.go - udp session manager
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

type getSessionRequest struct {
	addr     string
	response chan *session
}

/**
 * SessionManager emulates UDP "session" and manages them
 */
type sessionManager struct {
	sessions     map[string]*session
	scheduler    *scheduler.Scheduler
	statsHandler *stats.Handler
	sessionCount uint
	addSessionC  chan *session
	delSessionC  chan *session
	stopC        chan bool
	getSessionC  chan *getSessionRequest
}

/**
 * Creates new session manager
 */
func newSessionManager(scheduler *scheduler.Scheduler, statsHandler *stats.Handler) *sessionManager {
	return &sessionManager{
		scheduler:    scheduler,
		statsHandler: statsHandler,
		sessions:     make(map[string]*session),
		sessionCount: 0,
		addSessionC:  make(chan *session),
		delSessionC:  make(chan *session),
		stopC:        make(chan bool),
		getSessionC:  make(chan *getSessionRequest),
	}
}

/**
 * Sends buf received from serverConn, to backend on behalf of client, identified by clientAddr. Automatically selects backend.
 */
func (sm *sessionManager) Send(serverConn *net.UDPConn, clientAddr *net.UDPAddr, sessionTimeout time.Duration, udpResponses *int, buf []byte) error {

	if session, ok := sm.getSession(clientAddr); ok {
		session.send(buf)
		return nil
	}

	backend, err := sm.scheduler.TakeBackend(&core.UdpContext{
		RemoteAddr: *clientAddr,
	})

	if err != nil {
		return err
	}

	session := sm.createSession(clientAddr, sm.scheduler, backend)
	session.start(serverConn, sm, sessionTimeout, udpResponses)
	session.send(buf)

	return nil
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
			case session := <-sm.addSessionC:
				sm.sessionCount++
				sm.statsHandler.Connections <- sm.sessionCount
				sm.sessions[session.clientAddr.String()] = session

			/* Handle removig expired session */
			case session := <-sm.delSessionC:
				sm.sessionCount--
				sm.statsHandler.Connections <- sm.sessionCount
				delete(sm.sessions, session.clientAddr.String())

			/* Handle get session request */
			case request := <-sm.getSessionC:
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
func (sm *sessionManager) getSession(clientAddr *net.UDPAddr) (*session, bool) {
	request := &getSessionRequest{
		addr:     clientAddr.String(),
		response: make(chan *session),
	}
	sm.getSessionC <- request
	session := <-request.response
	return session, session != nil
}

/**
 * Adds session
 */
func (sm *sessionManager) add(session *session) {
	go func() {
		sm.addSessionC <- session
	}()
}

/**
 * Removes session
 */
func (sm *sessionManager) remove(session *session) {
	go func() {
		sm.delSessionC <- session
	}()
}

/**
 * Stops session manager
 */
func (sm *sessionManager) Stop() {
	go func() {
		sm.stopC <- true
	}()
}
