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

	/* Server name */
	name string

	/* Server configuration */
	cfg config.Server

	/* Scheduler */
	scheduler scheduler.Scheduler

	/* Session Manager */
	sessionManager *sessionManager

	/* Stats handler */
	statsHandler *stats.Handler

	/* Session timeout */
	sessionTimeout time.Duration

	/* ----- channels ----- */

	/* Stop channel */
	stop chan bool
}

/**
 * Creates new UDP server
 */
func NewUDPServer(name string, cfg config.Server) (*UDPServer, error) {

	log := logging.For("UDPServer")

	statsHandler := stats.NewHandler(name)

	server := &UDPServer{
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
		sessionTimeout: utils.ParseDurationOrDefault(*cfg.ClientIdleTimeout, DEFAULT_UDP_SESSION_IDLE_TIMEOUT),
	}

	if server.sessionTimeout == 0 {
		server.sessionTimeout = DEFAULT_UDP_SESSION_IDLE_TIMEOUT
	}

	log.Info("Creating UDP server '", name, "': ", cfg.Bind, " ", cfg.Balance, " ", cfg.Discovery.Kind, " ", cfg.Healthcheck.Kind)
	return server, nil
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
		log.Error("Error starting UDP Listen ", err)
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
	//serverConn.SetReadBuffer(UDP_PACKET_SIZE)

	if err != nil {
		log.Error("Error starting UDP server: ", err)
		return err
	}

	// Listen requests from clients
	var buf = make([]byte, UDP_PACKET_SIZE)
	maxResponses := this.cfg.MaxResponses

	// Main proxy loop goroutine
	go func() {
		for {
			n, clientAddr, err := serverConn.ReadFromUDP(buf)

			if err != nil {
				log.Error("Error ReadFromUDP: ", err)
				continue
			}

			go func(received []byte) {

				if session, ok := this.sessionManager.getForAddr(clientAddr); ok {
					session.sendToBackend(received)
					return
				}

				backend, err := this.scheduler.TakeBackend(&core.UdpContext{
					RemoteAddr: *clientAddr,
				})

				if err != nil {
					log.Error("Error TakeBackend: ", err)
					return
				}

				/* Store client by it's address+port, so that when we get responce from server, we could route it */
				log.Debug("Creating new UDP session for:", clientAddr.String())

				session := this.sessionManager.createSession(clientAddr, &this.scheduler, backend)
				session.start(serverConn, this.sessionManager, this.sessionTimeout, maxResponses)
				session.sendToBackend(received)
			}(buf[0:n])
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
