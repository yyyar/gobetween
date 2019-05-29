package session

import (
	"fmt"
	"net"
	"sync"
	"sync/atomic"
	"time"

	"github.com/yyyar/gobetween/core"
	"github.com/yyyar/gobetween/logging"
	"github.com/yyyar/gobetween/server/scheduler"
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

type packet struct {
	// pointer to object that has to be returned to buf pool
	payload []byte
	// length of the usable part of buffer
	len int
}

func (p packet) buf() []byte {
	if p.payload == nil {
		return nil
	}

	return p.payload[0:p.len]
}

func (p packet) release() {
	if p.payload == nil {
		return
	}
	bufPool.Put(p.payload)
}

type Session struct {
	//counters
	sent uint64
	recv uint64

	//session config
	cfg Config

	clientAddr *net.UDPAddr

	//connection to backend
	conn    net.Conn
	backend core.Backend

	//communication
	out     chan packet
	stopC   chan struct{}
	stopped uint32

	//scheduler
	scheduler *scheduler.Scheduler
}

func NewSession(clientAddr *net.UDPAddr, conn net.Conn, backend core.Backend, scheduler *scheduler.Scheduler, cfg Config) *Session {

	scheduler.IncrementConnection(backend)
	s := &Session{
		cfg:        cfg,
		clientAddr: clientAddr,
		conn:       conn,
		backend:    backend,
		scheduler:  scheduler,
		out:        make(chan packet, MAX_PACKETS_QUEUE),
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
			case pkt := <-s.out:
				if t != nil {
					if !t.Stop() {
						<-t.C
					}
					t.Reset(cfg.ClientIdleTimeout)
				}

				if pkt.payload == nil {
					panic("Program error, output channel should not be closed here")
				}

				n, err := s.conn.Write(pkt.buf())
				pkt.release()

				if err != nil {
					log.Errorf("Could not write data to udp connection: %v", err)
					break
				}

				if n != pkt.len {
					log.Errorf("Short write error: should write %d bytes, but %d written", pkt.len, n)
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
					case pkt := <-s.out:
						pkt.release()
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
	case s.out <- packet{dup, n}:
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
				if atomic.LoadUint32(&s.stopped) == 0 {
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
	return atomic.LoadUint32(&s.stopped) == 1
}

func (s *Session) Close() {
	select {
	case s.stopC <- struct{}{}:
	default:
	}
}
