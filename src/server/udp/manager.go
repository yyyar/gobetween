/**
 * manager.go - udp session manager
 *
 * @author Illarion Kovalchuk <illarion.kovalchuk@gmail.com>
 * @author Yaroslav Pogrebnyak <yyyaroslav@gmail.com>
 */

package udp

import (
	"../../config"
	"../../core"
	"../../stats"
	"../../utils"
	"../scheduler"
	"net"
)

type getSessionRequest struct {
	addr     string
	response chan *session
}

/**
 * SessionManager emulates UDP "session" and manages them
 */
type sessionManager struct {
	cfg          config.Server
	sessions     map[string]*session
	scheduler    *scheduler.Scheduler
	statsHandler *stats.Handler
	sessionCount uint
	addSessionC  chan *session
	delSessionC  chan *session
	getSessionC  chan *getSessionRequest
	stopC        chan bool
}

/**
 * Creates new session manager
 */
func newSessionManager(cfg config.Server, scheduler *scheduler.Scheduler, statsHandler *stats.Handler) *sessionManager {
	return &sessionManager{
		cfg:          cfg,
		scheduler:    scheduler,
		statsHandler: statsHandler,
		sessions:     make(map[string]*session),
		addSessionC:  make(chan *session),
		delSessionC:  make(chan *session),
		getSessionC:  make(chan *getSessionRequest),
		stopC:        make(chan bool),
	}
}

/**
 * Sends buf received from serverConn, to backend on behalf of client, identified by clientAddr. Automatically selects backend.
 */
func (sm *sessionManager) Send(serverConn *net.UDPConn, clientAddr *net.UDPAddr, buf []byte) error {

	if session, ok := sm.getSession(clientAddr); ok {
		return session.send(buf)
	}

	backend, err := sm.scheduler.TakeBackend(&core.UdpContext{
		RemoteAddr: *clientAddr,
	})

	if err != nil {
		return err
	}

	session, err := sm.createSession(clientAddr, serverConn, backend)

	if err != nil {
		return err
	}

	return session.send(buf)
}

/**
 * Creates new sessions; adds to itself and returns it
 */
func (sm *sessionManager) createSession(addr *net.UDPAddr, serverConn *net.UDPConn, backend *core.Backend) (*session, error) {

	udpResponses := 0
	if sm.cfg.Udp != nil {
		udpResponses = sm.cfg.Udp.MaxResponses
	}

	session := &session{
		clientIdleTimeout:  utils.ParseDurationOrDefault(*sm.cfg.ClientIdleTimeout, 0),
		backendIdleTimeout: utils.ParseDurationOrDefault(*sm.cfg.BackendIdleTimeout, 0),
		udpResponses:       udpResponses,
		sessionManager:     sm,
		scheduler:          sm.scheduler,
		serverConn:         serverConn,
		clientAddr:         addr,
		backend:            backend,
	}

	err := session.start()
	if err != nil {
		session.stop()
		return nil, err
	}

	sm.add(session)
	return session, nil
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
				sm.scheduler.IncrementConnection(*session.backend)
				sm.statsHandler.Connections <- sm.sessionCount
				sm.sessions[session.clientAddr.String()] = session

			/* Handle removig expired session */
			case session := <-sm.delSessionC:
				sm.sessionCount--
				sm.scheduler.DecrementConnection(*session.backend)
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
				return
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
