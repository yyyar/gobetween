/**
 * session.go - udp "session"
 *
 * @author Illarion Kovalchuk <illarion.kovalchuk@gmail.com>
 * @author Yaroslav Pogrebnyak <yyyaroslav@gmail.com>
 */
package udp

import (
	"net"
	"sync/atomic"
	"time"

	"../../core"
	"../../logging"
	"../scheduler"
)

/**
 * Emulates UDP "session"
 */
type session struct {

	/* timeout for new data from client */
	clientIdleTimeout time.Duration

	/* timeout for new data from backend */
	backendIdleTimeout time.Duration

	/* max number of client requests */
	maxRequests uint64

	/* actually sent client requests */
	_sentRequests uint64

	/* max number of backend responses */
	maxResponses uint64

	/* scheduler */
	scheduler *scheduler.Scheduler

	/* connection to send responses to client with */
	serverConn *net.UDPConn

	/* client address */
	clientAddr net.UDPAddr

	/* Session backend */
	backend *core.Backend

	/* connection to previously elected backend */
	backendConn *net.UDPConn

	/* activity channel */
	clientActivityC chan bool

	clientLastActivity time.Time

	/* stop channel */
	stopC chan bool

	/* function to call to notify server that session is closed and should be removed */
	notifyClosed func()
}

/**
 * Start session
 */
func (s *session) start() error {

	log := logging.For("udp/Session")

	s.stopC = make(chan bool)
	s.clientActivityC = make(chan bool)
	s.clientLastActivity = time.Now()

	backendAddr, err := net.ResolveUDPAddr("udp", s.backend.Target.String())

	if err != nil {
		log.Error("Error ResolveUDPAddr: ", err)
		return err
	}

	backendConn, err := net.DialUDP("udp", nil, backendAddr)

	if err != nil {
		log.Debug("Error connecting to backend: ", err)
		return err
	}

	s.backendConn = backendConn

	/**
	 * Update time and wait for stop
	 */
	var t *time.Ticker
	var tC <-chan time.Time

	if s.clientIdleTimeout > 0 {
		log.Debug("Starting new ticker for client ", s.clientAddr, " ", s.clientIdleTimeout)
		t = time.NewTicker(s.clientIdleTimeout)
		tC = t.C
	}

	stopped := false
	go func() {
		for {
			select {
			case now := <-tC:
				if s.clientLastActivity.Add(s.clientIdleTimeout).Before(now) {
					log.Debug("Client ", s.clientAddr, " was idle for more than ", s.clientIdleTimeout)
					go func() {
						s.stopC <- true
					}()
				}
			case <-s.stopC:
				stopped = true
				log.Debug("Closing client session: ", s.clientAddr)
				s.backendConn.Close()
				s.notifyClosed()
				if t != nil {
					t.Stop()
				}
				return
			case <-s.clientActivityC:
				s.clientLastActivity = time.Now()
			}
		}
	}()

	/**
	 * Proxy data from backend to client
	 */
	go func() {
		buf := make([]byte, UDP_PACKET_SIZE)
		var responses uint64

		for {
			if s.backendIdleTimeout > 0 {
				err := s.backendConn.SetReadDeadline(time.Now().Add(s.backendIdleTimeout))
				if err != nil {
					log.Error("Unable to set timeout for backend connection, closing. Error: ", err)
					s.stop()
					return
				}
			}
			n, _, err := s.backendConn.ReadFromUDP(buf)

			if err != nil {

				if !err.(*net.OpError).Timeout() && !stopped {
					log.Error("Error reading from backend ", err)
				}

				s.stop()
				return
			}

			s.scheduler.IncrementRx(*s.backend, uint(n))
			s.serverConn.WriteToUDP(buf[0:n], &s.clientAddr)

			if s.maxResponses > 0 {
				responses++
				if responses >= s.maxResponses {
					s.stop()
					return
				}
			}
		}
	}()
	return nil
}

/**
 * Writes data to session backend
 */
func (s *session) send(buf []byte) error {
	select {
	case s.clientActivityC <- true:
	default:
	}

	_, err := s.backendConn.Write(buf)
	if err != nil {
		return err
	}

	s.scheduler.IncrementTx(*s.backend, uint(len(buf)))

	if s.maxRequests > 0 {
		if atomic.AddUint64(&s._sentRequests, 1) >= s.maxRequests {
			s.stop()
		}
	}

	return nil
}

/**
 * Stops session
 */
func (c *session) stop() {
	select {
	case c.stopC <- true:
	default:
	}
}
