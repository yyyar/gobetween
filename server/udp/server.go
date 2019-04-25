package udp

/**
 * server.go - UDP server implementation
 *
 * @author Illarion Kovalchuk <illarion.kovalchuk@gmail.com>
 */

import (
	"fmt"
	"net"
	"sync"
	"sync/atomic"
	"time"

	"github.com/yyyar/gobetween/balance"
	"github.com/yyyar/gobetween/config"
	"github.com/yyyar/gobetween/core"
	"github.com/yyyar/gobetween/discovery"
	"github.com/yyyar/gobetween/healthcheck"
	"github.com/yyyar/gobetween/logging"
	"github.com/yyyar/gobetween/server/modules/access"
	"github.com/yyyar/gobetween/server/scheduler"
	"github.com/yyyar/gobetween/server/udp/session"
	"github.com/yyyar/gobetween/stats"
	"github.com/yyyar/gobetween/utils"
	"github.com/eric-lindau/udpfacade"
)

const UDP_PACKET_SIZE = 65507
const CLEANUP_EVERY = time.Second * 2

var log = logging.For("udp/server")

/**
 * UDP server implementation
 */
type Server struct {
	/* Server name */
	name string

	/* Server configuration */
	cfg config.Server

	/* Scheduler */
	scheduler *scheduler.Scheduler

	/* Server connection */
	serverConn *net.UDPConn

	/* Flag indicating that server is stopped */
	stopped uint32

	/* Stop channel */
	stop chan bool

	/* ----- modules ----- */

	/* Access module checks if client is allowed to connect */
	access *access.Access

	/* ----- sessions ----- */
	sessions map[string]*session.Session
	mu       sync.Mutex
}

/**
 * Creates new UDP server
 */
func New(name string, cfg config.Server) (*Server, error) {

	statsHandler := stats.NewHandler(name)
	scheduler := &scheduler.Scheduler{
		Balancer:     balance.New(nil, cfg.Balance),
		Discovery:    discovery.New(cfg.Discovery.Kind, *cfg.Discovery),
		Healthcheck:  healthcheck.New(cfg.Healthcheck.Kind, *cfg.Healthcheck),
		StatsHandler: statsHandler,
	}

	server := &Server{
		name:      name,
		cfg:       cfg,
		scheduler: scheduler,
		stop:      make(chan bool),
		sessions:  make(map[string]*session.Session),
	}

	/* Add access if needed */
	if cfg.Access != nil {
		access, err := access.NewAccess(cfg.Access)
		if err != nil {
			return nil, fmt.Errorf("Could not initialize access restrictions: %v", err)
		}
		server.access = access
	}

	log.Info("Creating UDP server '", name, "': ", cfg.Bind, " ", cfg.Balance, " ", cfg.Discovery.Kind, " ", cfg.Healthcheck.Kind)
	return server, nil
}

/**
 * Returns current server configuration
 */
func (this *Server) Cfg() config.Server {
	return this.cfg
}

/**
 * Starts server
 */
func (this *Server) Start() error {

	// Start listening
	if err := this.listen(); err != nil {
		return fmt.Errorf("Could not start listening UDP: %v", err)
	}

	this.scheduler.StatsHandler.Start()
	this.scheduler.Start()
	this.serve()

	go func() {

		ticker := time.NewTicker(CLEANUP_EVERY)

		for {
			select {
			case <-ticker.C:
				this.cleanup()
				/* handle server stop */
			case <-this.stop:
				log.Info("Stopping ", this.name)
				atomic.StoreUint32(&this.stopped, 1)

				ticker.Stop()

				this.serverConn.Close()

				this.scheduler.StatsHandler.Stop()
				this.scheduler.Stop()

				this.mu.Lock()
				for k, s := range this.sessions {
					delete(this.sessions, k)
					s.Close()
				}
				this.mu.Unlock()

				return
			}
		}
	}()

	return nil
}

/**
 * Start accepting connections
 */
func (this *Server) listen() error {
	listenAddr, err := net.ResolveUDPAddr("udp", this.cfg.Bind)
	if err != nil {
		return fmt.Errorf("Failed to resolve udp address %v : %v", this.cfg.Bind, err)
	}

	this.serverConn, err = net.ListenUDP("udp", listenAddr)

	if err != nil {
		return fmt.Errorf("Failed to create listening udp socket: %v", err)
	}

	return nil
}

/**
 * Start serving
 */
func (this *Server) serve() {

	cfg := session.Config{
		MaxRequests:        this.cfg.Udp.MaxRequests,
		MaxResponses:       this.cfg.Udp.MaxResponses,
		ClientIdleTimeout:  utils.ParseDurationOrDefault(*this.cfg.ClientIdleTimeout, 0),
		BackendIdleTimeout: utils.ParseDurationOrDefault(*this.cfg.BackendIdleTimeout, 0),
		Transparent:        this.cfg.Udp.Transparent,
	}

	// Main loop goroutine - reads incoming data and decides what to do
	go func() {

		buf := make([]byte, UDP_PACKET_SIZE)
		for {
			n, clientAddr, err := this.serverConn.ReadFromUDP(buf)

			if err != nil {
				if atomic.LoadUint32(&this.stopped) == 1 {
					return
				}

				log.Error("Failed to read from UDP: ", err)

				continue
			}

			if this.access != nil {
				if !this.access.Allows(&clientAddr.IP) {
					log.Debug("Client disallowed to connect: ", clientAddr.IP)
					continue
				}
			}

			//special case for single request mode
			if cfg.MaxRequests == 1 {
				err := this.fireAndForget(clientAddr, buf[:n])

				if err != nil {
					log.Errorf("Error sending data to backend: %v ", err)
				}

				continue
			}

			this.proxy(cfg, clientAddr, buf[:n])

		}
	}()
}

/**
 * Safely remove connections that have marked themself as done
 */
func (this *Server) cleanup() {
	this.mu.Lock()
	defer this.mu.Unlock()

	for k, s := range this.sessions {
		if s.IsDone() {
			delete(this.sessions, k)
		}

	}

	this.scheduler.StatsHandler.Connections <- uint(len(this.sessions))
}

/**
 * Elect and connect to backend
 */
func (this *Server) electAndConnect(clientAddr *net.UDPAddr) (net.Conn, *core.Backend, error) {
	backend, err := this.scheduler.TakeBackend(core.UdpContext{
		ClientAddr: *clientAddr,
	})

	if err != nil {
		return nil, nil, fmt.Errorf("Could not elect backend for clientAddr %v: %v", clientAddr, err)
	}

	host := backend.Host
	port := backend.Port

	addrStr := host + ":" + port

	addr, err := net.ResolveUDPAddr("udp", addrStr)
	if err != nil {
		return nil, nil, fmt.Errorf("Could not resolve udp address %s: %v", addrStr, err)
	}

	var conn net.Conn
	if this.cfg.Udp.Transparent {
		conn, err = udpfacade.DialUDPFrom(clientAddr, addr)
		if err != nil {
			return nil, nil, fmt.Errorf("Could not dial UDP addr %v from %v: %v", addr, clientAddr, err)
		}
	} else {
		conn, err = net.DialUDP("udp", nil, addr)
		if err != nil {
			return nil, nil, fmt.Errorf("Could not dial UDP addr %v: %v", addr, err)
		}
	}

	return conn, backend, nil
}

/**
 * Get or create session
 */
func (this *Server) getOrCreateSession(cfg session.Config, clientAddr *net.UDPAddr) (*session.Session, error) {
	key := clientAddr.String()

	this.mu.Lock()
	defer this.mu.Unlock()

	s, ok := this.sessions[key]

	//session exists and is not done yet
	if ok && !s.IsDone() {
		return s, nil
	}

	//session exists but should be replaced with a new one
	if ok {
		go func() { s.Close() }()
	}

	conn, backend, err := this.electAndConnect(clientAddr)
	if err != nil {
		return nil, fmt.Errorf("Could not elect/connect to backend: %v", err)
	}

	s = session.NewSession(clientAddr, conn, *backend, this.scheduler, cfg)
	if !cfg.Transparent {
		s.ListenResponses(this.serverConn)
	}

	this.sessions[key] = s

	this.scheduler.StatsHandler.Connections <- uint(len(this.sessions))

	return s, nil
}

/**
 * Get the session and send data via chosen session
 */
func (this *Server) proxy(cfg session.Config, clientAddr *net.UDPAddr, buf []byte) {

	s, err := this.getOrCreateSession(cfg, clientAddr)
	if err != nil {
		log.Error(err)
		return
	}

	err = s.Write(buf)
	if err != nil {
		log.Errorf("Could not write data to UDP 'session' %v: %v", s, err)
		return
	}

}

/**
 * Omit creating session, just send one packet of data
 */
func (this *Server) fireAndForget(clientAddr *net.UDPAddr, buf []byte) error {

	conn, backend, err := this.electAndConnect(clientAddr)
	if err != nil {
		return fmt.Errorf("Could not elect or connect to backend: %v", err)
	}

	n, err := conn.Write(buf)
	if err != nil {
		return fmt.Errorf("Could not write data to %v: %v", clientAddr, err)
	}

	if n != len(buf) {
		return fmt.Errorf("Failed to send full packet, expected size %d, actually sent %d", len(buf), n)
	}

	this.scheduler.IncrementTx(*backend, uint(n))

	return nil

}

/**
 * Stop, dropping all connections
 */
func (this *Server) Stop() {
	this.stop <- true
}
