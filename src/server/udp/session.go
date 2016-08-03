package udp

import (
	"../../core"
	"../../logging"
	"../../server"
	"../../stats"
	"net"
	"time"
)

type sessionManager struct {
	statsHandler *stats.Handler
	sessions     map[string]*session
	sessionCount uint
	addC         chan *session
	remC         chan *session
	stopC        chan bool
}

func (sm *sessionManager) add(session *session) {
	go func() {
		sm.addC <- session
	}()
}

func (sm *sessionManager) remove(session *session) {
	go func() {
		sm.remC <- session
	}()
}

func (sm *sessionManager) start() {
	go func() {
		for {
			select {
			case session := <-sm.addC:
				sm.sessionCount++
				sm.statsHandler.Connections <- sm.sessionCount
				sm.sessions[session.clientAddr.String()] = session
			case session := <-sm.remC:
				sm.sessionCount--
				sm.statsHandler.Connections <- sm.sessionCount
				delete(sm.sessions, session.clientAddr.String())
			case <-sm.stopC:
				for _, session := range sm.sessions {
					session.stop()
				}
			}

		}
	}()
}

func (sm *sessionManager) stop() {
	go func() {
		sm.stopC <- true
	}()
}

func (sm *sessionManager) getForAddr(clientAddr *net.UDPAddr) (*session, bool) {
	result, ok := sm.sessions[clientAddr.String()]
	return result, ok
}

func newSessionManager(statsHandler *stats.Handler) *sessionManager {
	return &sessionManager{
		statsHandler: statsHandler,
		sessions:     make(map[string]*session),
		sessionCount: 0,
		addC:         make(chan *session),
		remC:         make(chan *session),
		stopC:        make(chan bool),
	}
}

type session struct {
	clientAddr   *net.UDPAddr // Address of client
	statsHandler *stats.Handler
	scheduler    *server.Scheduler
	backend      *core.Backend
	backendConn  *net.UDPConn // Connection to backend created for this client
	lastUpdated  time.Time

	updC    chan bool
	stopC   chan bool
}

func (c *session) start(serverConn *net.UDPConn, sessionManager *sessionManager, timeout time.Duration) {
	log := logging.For("udp session")

	go func() {

		ticker := time.NewTicker(timeout)
		for {
			select {
			case <-c.stopC:
				ticker.Stop()
				c.scheduler.DecrementConnection(*c.backend)
				c.backendConn.Close()
				sessionManager.remove(c)
			case <-c.updC:
				c.lastUpdated = time.Now()
			case now := <-ticker.C:
				if c.lastUpdated.Add(timeout).Before(now) {
					c.stop()
				}
			}
		}
	}()

	go func() {
		var buf = make([]byte, UDP_PACKET_SIZE)
		for {
			n, _, err := c.backendConn.ReadFromUDP(buf)
			if err != nil {
				log.Debug("Closing client ", c.clientAddr.String())
				break
			}
			c.markUpdated()
			c.scheduler.IncrementRx(*c.backend, uint(n))
			serverConn.WriteToUDP(buf[0:n], c.clientAddr)
			c.scheduler.IncrementTx(*c.backend, uint(n))
		}
	}()
}

func (c *session) markUpdated() {
	go func(){
		c.updC <- true
	}()
}

func (c *session) sendToBackend(buf []byte) {
	go func() {
		c.backendConn.Write(buf)
		c.markUpdated()
		n := len(buf)
		c.scheduler.IncrementTx(*c.backend, uint(n))
	}()
}

func (c *session) stop() {
	go func() {
		c.stopC <- true
	}()
}

func newSession(addr *net.UDPAddr, statsHandler *stats.Handler, scheduler *server.Scheduler, backend *core.Backend, backendConn *net.UDPConn) *session {
	scheduler.IncrementConnection(*backend)
	return &session{
		clientAddr:   addr,
		statsHandler: statsHandler,
		scheduler:    scheduler,
		backend:      backend,
		backendConn:  backendConn,
		lastUpdated:  time.Now(),
		updC:         make(chan bool),
		stopC:        make(chan bool),
	}
}
