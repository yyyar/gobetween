package session

import (
	"../../../core"
	"../../../logging"
	"../../scheduler"
	"fmt"
	"net"
	"sync"
	"sync/atomic"
	"time"
)

const (
	UDP_PACKET_SIZE   = 65507
	MAX_PACKETS_QUEUE = 10000
)

var log = logging.For("udp/server/session")
var bufPool = sync.Pool{
	New: func() interface{} {
		return make([]byte, UDP_PACKET_SIZE)
	},
}

type Session struct {
	//counters
	sent uint64
	recv uint64

	//session config
	cfg Config

	clientAddr *net.UDPAddr

	//connection to backend
	conn    *net.UDPConn
	backend core.Backend

	//communication
	out     chan []byte
	stopC   chan struct{}
	stopped uint32

	//scheduler
	scheduler *scheduler.Scheduler
}

func NewSession(clientAddr *net.UDPAddr, conn *net.UDPConn, backend core.Backend, scheduler *scheduler.Scheduler, cfg Config) *Session {

	scheduler.IncrementConnection(backend)
	s := &Session{
		cfg:        cfg,
		clientAddr: clientAddr,
		conn:       conn,
		backend:    backend,
		scheduler:  scheduler,
		out:        make(chan []byte, MAX_PACKETS_QUEUE),
		stopC:      make(chan struct{}, 1),
	}

	go func() {

		var t *time.Timer
		var tC <-chan time.Time

		if cfg.ClientIdleTimeout > 0 {
			t = time.NewTimer(cfg.ClientIdleTimeout)
			tC = t.C
		}

		for {
			select {

			case <-tC:
				s.Close()
			case buf := <-s.out:
				if t != nil {
					if !t.Stop() {
						<-t.C
					}
					t.Reset(cfg.ClientIdleTimeout)
				}

				if buf == nil {
					panic("Program error, output channel should not be closed here")
				}

				n, err := s.conn.Write(buf)
				bufPool.Put(buf)

				if err != nil {
					log.Errorf("Could not write data to udp connection: %v", err)
					break
				}

				if n != len(buf) {
					log.Errorf("Short write error: should write %d bytes, but %d written", len(buf), n)
					break
				}

				s.scheduler.IncrementTx(s.backend, uint(n))

				if s.cfg.MaxRequests > 0 && atomic.AddUint64(&s.sent, 1) > s.cfg.MaxRequests {
					log.Errorf("Restricted to send more UDP packets")
					break
				}
			case <-s.stopC:
				atomic.StoreUint32(&s.stopped, 1)
				if t != nil {
					t.Stop()
				}
				s.conn.Close()
				s.scheduler.DecrementConnection(s.backend)
				// drain output packets channel and free buffers
				for {
					select {
					case buf := <-s.out:
						bufPool.Put(buf)
					default:
						return
					}
				}

			}
		}

	}()

	return s
}

func (s *Session) Write(buf []byte) error {
	if atomic.LoadUint32(&s.stopped) == 1 {
		return fmt.Errorf("Closed session")
	}

	dup := bufPool.Get().([]byte)
	n := copy(dup, buf)

	select {
	case s.out <- dup[0:n]:
	default:
		bufPool.Put(dup)
	}

	return nil
}

/**
 * ListenResponses waits for responses from backend, and sends them back to client address via
 * server connection, so that client is not confused with source host:port of the
 * packet it receives
 */
func (s *Session) ListenResponses(sendTo *net.UDPConn) {

	go func() {
		b := make([]byte, UDP_PACKET_SIZE)

		defer s.Close()

		for {

			if s.cfg.BackendIdleTimeout > 0 {
				s.conn.SetReadDeadline(time.Now().Add(s.cfg.BackendIdleTimeout))
			}

			n, err := s.conn.Read(b)

			if err != nil {
				/* if a backendidletimeout is reached for udp, this is pretty normal
				 * although does throw an 'error' technically.. for now i'll comment it out
				 * if atomic.LoadUint32(&s.stopped) == 0 {
				 *	log.Errorf("Failed to read from backend: %v", err)
				 * }
				 */
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
	return atomic.LoadUint32(&s.stopped) == 1
}

func (s *Session) Close() {
	select {
	case s.stopC <- struct{}{}:
	default:
	}
}
