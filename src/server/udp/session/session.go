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
	UDP_PACKET_SIZE		= 65507
	MAX_PACKETS_QUEUE	= 10000
)

var bufPool = sync.Pool{
	New: func() interface{} {
		return make([]byte, UDP_PACKET_SIZE)
	},
}

type sendBufPool struct {
	buf		[]byte
	bufsize		int
}

type BackendSession struct {
	//counters
	sent uint64
	recv uint64

	//session config
	cfg Config

	clientAddr	*net.UDPAddr

	//connection to backend
	backendconn	*net.UDPConn
	backend		core.Backend

	//communication
	out		chan sendBufPool
	stopC		chan struct{}
	stopped		uint32

	//scheduler
	scheduler	*scheduler.Scheduler
}

func NewSession(clientAddr *net.UDPAddr, backendconn *net.UDPConn, backend core.Backend, scheduler *scheduler.Scheduler, cfg Config) *BackendSession {
	log := logging.For("udp/server/session/NewSession")
	log.Debug("backend ", backend.Target.Address(), ": client [", clientAddr.IP, "]")

	scheduler.IncrementConnection(backend)
	s := &BackendSession{
		cfg:		cfg,
		clientAddr:	clientAddr,
		backendconn:	backendconn,
		backend:	backend,
		scheduler:	scheduler,
		out:		make(chan sendBufPool, MAX_PACKETS_QUEUE),
		stopC:		make(chan struct{}, 1),
	}

	go func() {

		var t *time.Timer
		var tC <-chan time.Time

		if cfg.ClientIdleTimeout > 0 {
			t = time.NewTimer(cfg.ClientIdleTimeout)
			tC = t.C
		}

		// wait for; the idle timer, output to send to the backend, or a close request
		for {
			select {

			case <-tC:
				s.BackendClose()

			case sendBufPool := <-s.out:
				if t != nil {
					if !t.Stop() {
						<-t.C
					}
					t.Reset(cfg.ClientIdleTimeout)
				}

				if sendBufPool.buf == nil {
					panic("Program error, output channel should not be closed here")
				}

				log.Debug("backend ", backend.Target.Address(), ": client [", clientAddr.IP, "], backend send, bytes ", sendBufPool.bufsize)
				n, err := s.backendconn.Write(sendBufPool.buf[0:sendBufPool.bufsize])
				bufPool.Put(sendBufPool.buf)

				if err != nil {
					log.Errorf("Could not send data to udp connection: %v", err)
					break
				}

				if n != len(sendBufPool.buf[0:sendBufPool.bufsize]) {
					log.Errorf("short send error: should write %d bytes, but %d written", len(sendBufPool.buf[0:sendBufPool.bufsize]), n)
					break
				}

				s.scheduler.IncrementTx(s.backend, uint(n))

				if s.cfg.MaxRequests > 0 && atomic.AddUint64(&s.sent, 1) > s.cfg.MaxRequests {
					log.Errorf("MaxRequests %d reached, will not send more UDP packets", s.cfg.MaxRequests)
					break
				}

			case <-s.stopC:
				atomic.StoreUint32(&s.stopped, 1)
				if t != nil {
					t.Stop()
				}
				s.backendconn.Close()
				s.scheduler.DecrementConnection(s.backend)
				// drain output packets channel and put buffers back in pool
				for {
					select {
					case sendBufPool := <-s.out:
						bufPool.Put(sendBufPool.buf)
					default:
						return
					}
				}

			}
		}

	}()

	return s
}

func (s *BackendSession) BackendSend(buf []byte) error {
	log := logging.For("udp/server/session/BackendSend")
	log.Debug("backend ", s.backend.Target.Address(), ": client [", s.clientAddr.IP, "], backend send, bytes ", len(buf))

	if atomic.LoadUint32(&s.stopped) == 1 {
		return fmt.Errorf("Closed session")
	}

	dup := bufPool.Get().([]byte)
	n := copy(dup, buf)

	sendBufPool := sendBufPool{
		buf:		dup,
		bufsize:	n,
	}

	// if the backend session channel is listening, use that, otherwise dump it
	select {
		case s.out <- sendBufPool:
			//log.Debug("sending to channel ",len(sendBufPool.buf), " bytes, size ", n);

		default:
			bufPool.Put(dup)
			//log.Debug("could not send data to session channel");
	}

	return nil
}

/**
 * BackendListenAndRelayToClient waits for responses from backend, and sends them back 
 * to client address via server connection. This means the client is not confused with source 
 * host:port of the packet it receives
 */
func (s *BackendSession) BackendListenAndRelayToClient(sendTo *net.UDPConn) {
	log := logging.For("udp/server/session/BackendListenAndRelayToClient")
	log.Debug("backend ", s.backend.Target.Address(), ": client [", s.clientAddr.IP, "]")

	go func() {
		b := make([]byte, UDP_PACKET_SIZE)

		defer s.BackendClose()

		for {

			if s.cfg.BackendIdleTimeout > 0 {
				s.backendconn.SetReadDeadline(time.Now().Add(s.cfg.BackendIdleTimeout))
			}

			n, err := s.backendconn.Read(b)
			log.Debug("backend ", s.backend.Target.Address(), ": client [", s.clientAddr.IP, "], backend receive, bytes ", len(b[:n]))

			if err != nil {
				 if atomic.LoadUint32(&s.stopped) == 0 {
				 	log.Errorf("Failed to receive from backend: %v", err)
				 }
				return
			}

			s.scheduler.IncrementRx(s.backend, uint(n))

			log.Debug("backend ", s.backend.Target.Address(), ": client [", s.clientAddr.IP, "], client send, bytes ", len(b[:n]))
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

func (s *BackendSession) IsDone() bool {
	//log := logging.For("udp/server/session/IsDone")
	//log.Debug("client ", s.clientAddr)

	return atomic.LoadUint32(&s.stopped) == 1
}

func (s *BackendSession) BackendClose() {
	log := logging.For("udp/server/session/BackendClose")
	log.Debug("backend ", s.backend.Target.Address(), ": client [", s.clientAddr.IP, "]")

	select {
	case s.stopC <- struct{}{}:
	default:
	}
}
