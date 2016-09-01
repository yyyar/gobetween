/**
 * udpserver.go - UDP server implementation
 *
 * @author Illarion Kovalchuk <illarion.kovalchuk@gmail.com>
 */

package udp

import (
	"../../balance"
	"../../config"
	"../../core"
	"../../discovery"
	"../../healthcheck"
	"../../logging"
	"../../stats"
	"../../utils"
	"../scheduler"
	"net"
	"time"
)

const (
	UDP_PACKET_SIZE                  = 65507
	DEFAULT_UDP_SESSION_IDLE_TIMEOUT = time.Minute * 1
)

/**
 * UDP server implementation
 */
type UDPServer struct {
	name           string
	cfg            config.Server
	scheduler      scheduler.Scheduler
	sessionManager *sessionManager
	statsHandler   *stats.Handler
	stop           chan bool
	idleTimeout    time.Duration
}

/**
 * Creates new UDP server
 */
func NewUDPServer(name string, cfg config.Server) (*UDPServer, error) {

	log := logging.For("UDPServer")

	idleTimeout := utils.ParseDurationOrDefault(*cfg.ClientIdleTimeout, DEFAULT_UDP_SESSION_IDLE_TIMEOUT)

	statsHandler := stats.NewHandler(name)

	udpServer := &UDPServer{
		name: name,
		cfg:  cfg,
		scheduler: scheduler.Scheduler{
			Balancer:     balance.New(cfg.Balance),
			Discovery:    discovery.New(cfg.Discovery.Kind, *cfg.Discovery),
			Healthcheck:  healthcheck.New(cfg.Healthcheck.Kind, *cfg.Healthcheck),
			StatsHandler: statsHandler,
		},
		sessionManager: newSessionManager(statsHandler),
		statsHandler:   statsHandler,
		stop:           make(chan bool),
		idleTimeout:    idleTimeout,
	}

	log.Info("Creating UDP '", name, "': ", cfg.Bind, " ", cfg.Balance, " ", cfg.Discovery.Kind, " ", cfg.Healthcheck.Kind)
	return udpServer, nil
}

/**
 * Returns current server configuration
 */
func (this *UDPServer) Cfg() config.Server {
	return this.cfg
}

/**
 * Starts server
 */
func (this *UDPServer) Start() error {
	log := logging.For("UDPServer.Listen")
	this.statsHandler.Start()
	this.scheduler.Start()
	this.sessionManager.start()

	go func() {
		for {
			select {

			case <-this.stop:
				this.sessionManager.stop()
				this.scheduler.Stop()
				this.statsHandler.Stop()
				return
			}
		}
	}()

	// Start listening
	if err := this.Listen(); err != nil {
		this.Stop()
		log.Error("Error starting Listen", err)
		return err
	}
	return nil
}

/**
 * Start accepting connections
 */
func (this *UDPServer) Listen() error {
	log := logging.For("UDPServer.Listen")

	listenAddr, err := net.ResolveUDPAddr("udp", this.cfg.Bind)
	serverConn, err := net.ListenUDP("udp", listenAddr)

	if err != nil {
		log.Error("Error starting UDP server: ", err)
		return err
	}

	var sessionTimeout time.Duration
	if this.idleTimeout == 0 {
		sessionTimeout = DEFAULT_UDP_SESSION_IDLE_TIMEOUT
	} else {
		sessionTimeout = this.idleTimeout
	}

	// Listen requests from clients
	var buf = make([]byte, UDP_PACKET_SIZE)
	go func() {
		for {
			log.Debug("Waiting for packet from clients")
			n, clientAddr, err := serverConn.ReadFromUDP(buf)

			if err != nil {
				log.Error("Error ReadFromUDP: ", err)
				continue
			}

			if session, ok := this.sessionManager.getForAddr(clientAddr); ok {
				session.sendToBackend(buf[0:n])
				continue
			}

			backend, err := this.scheduler.TakeBackend(&core.UdpContext{
				RemoteAddr: *clientAddr,
			})

			if err != nil {
				log.Error("Error TakeBackend: ", err)
				continue
			}

			backendAddr, err := net.ResolveUDPAddr("udp", backend.Target.String())
			if err != nil {
				log.Error("Error ResolveUDPAddr: ", err)
				continue
			}

			backendConn, err := net.DialUDP("udp", nil, backendAddr)

			if err != nil {
				log.Debug("Error connecting to backend: ", err)
				continue
			}

			//store client by it's address+port, so that when we get responce from server, we could route it
			log.Debug("Creating new session for:", clientAddr.String())
			session := this.sessionManager.createSession(clientAddr, this.statsHandler, &this.scheduler, backend, backendConn)
			session.start(serverConn, this.sessionManager, sessionTimeout)
			session.sendToBackend(buf[0:n])

		}
	}()

	return nil
}

/**
 * Stop, dropping all connections
 */
func (this *UDPServer) Stop() {
	log := logging.For("server.Listen")
	log.Info("Stopping ", this.name)
	this.stop <- true
}
