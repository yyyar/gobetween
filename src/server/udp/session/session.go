package session

import (
	"../../../core"
	"../../../logging"
	"../../scheduler"
	"fmt"
	"net"
	"sync/atomic"
	"time"
)

const UDP_PACKET_SIZE = 65507

type Session struct {
	/*atomics*/
	sent uint64
	recv uint64
	done uint32
	/*session config*/
	cfg Config

	lastClientActivity time.Time
	clientAddr         *net.UDPAddr

	//connection to backend
	conn    *net.UDPConn
	backend core.Backend

	//scheduler
	scheduler *scheduler.Scheduler
}

func NewSession(clientAddr *net.UDPAddr, conn *net.UDPConn, backend core.Backend, scheduler *scheduler.Scheduler, cfg Config) *Session {
	scheduler.IncrementConnection(backend)
	return &Session{
		cfg:                cfg,
		clientAddr:         clientAddr,
		conn:               conn,
		backend:            backend,
		lastClientActivity: time.Now(),
		scheduler:          scheduler,
	}
}

func (s *Session) Write(buf []byte) error {
	s.lastClientActivity = time.Now()

	n, err := s.conn.Write(buf)

	if err != nil {
		return fmt.Errorf("Could not write data to udp connection: %v", err)
	}

	if n != len(buf) {
		return fmt.Errorf("Short write error: should write %d bytes, but %d written", len(buf), n)
	}

	s.scheduler.IncrementTx(s.backend, uint(n))

	if s.cfg.MaxRequests > 0 && atomic.AddUint64(&s.sent, 1) > s.cfg.MaxRequests {
		atomic.StoreUint32(&s.done, 1)
		return fmt.Errorf("Restricted to send more UDP packets")
	}

	return nil
}

/**
 * ListenResponses waits for responses from backend, and sends them back to client address via
 * server connection, so that client is not confused with source host:port of the
 * packet it receives
 */
func (s *Session) ListenResponses(sendTo *net.UDPConn) {

	log := logging.For("udp/server/session")

	go func() {
		defer atomic.StoreUint32(&s.done, 1)

		b := make([]byte, UDP_PACKET_SIZE)

		for {

			if s.cfg.BackendIdleTimeout > 0 {
				s.conn.SetReadDeadline(time.Now().Add(s.cfg.BackendIdleTimeout))
			}

			n, err := s.conn.Read(b)

			if err != nil {
				if atomic.CompareAndSwapUint32(&s.done, 0, 1) {
					log.Errorf("Failed to read from backend: %v", err)
				}
				return
			}

			s.scheduler.IncrementRx(s.backend, uint(n))

			m, err := sendTo.WriteToUDP(b[0:n], s.clientAddr)

			if err != nil {
				log.Errorf("Could not send backend response to client: %v", err)
				return
			}

			if m != n {
				return
			}

			if s.cfg.MaxResponses > 0 && atomic.AddUint64(&s.recv, 1) >= s.cfg.MaxResponses {
				return
			}
		}
	}()
}

func (s *Session) IsDone() bool {
	if s.cfg.ClientIdleTimeout > 0 && s.lastClientActivity.Add(s.cfg.ClientIdleTimeout).Before(time.Now()) {
		atomic.StoreUint32(&s.done, 1)
		return true
	}
	return atomic.LoadUint32(&s.done) == 1
}

func (s *Session) CloseConn() {
	atomic.StoreUint32(&s.done, 1)
	s.conn.Close()
	s.scheduler.DecrementConnection(s.backend)
}
