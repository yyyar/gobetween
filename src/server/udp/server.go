/**
 * server.go - UDP server implementation
 *
 * @author Illarion Kovalchuk <illarion.kovalchuk@gmail.com>
 */

package udp

import (
	"../../balance"
	"../../config"
	"../../discovery"
	"../../healthcheck"
	"../../logging"
	"../../stats"
	"../../utils"
	"../scheduler"
	"net"
	"time"
)

const UDP_PACKET_SIZE = 65507

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

	/* Session Manager */
	sessionManager *sessionManager

	/* Stats handler */
	statsHandler *stats.Handler

	/* Session timeout */
	sessionTimeout time.Duration

	/* Server connection */
	serverConn *net.UDPConn

	/* ----- channels ----- */

	/* Stop channel */
	stop chan bool

	stopped bool
}

/**
 * Creates new UDP server
 */
func New(name string, cfg config.Server) (*Server, error) {

	log := logging.For("UDPServer")

	statsHandler := stats.NewHandler(name)
	scheduler := &scheduler.Scheduler{
		Balancer:     balance.New(cfg.Balance),
		Discovery:    discovery.New(cfg.Discovery.Kind, *cfg.Discovery),
		Healthcheck:  healthcheck.New(cfg.Healthcheck.Kind, *cfg.Healthcheck),
		StatsHandler: statsHandler,
	}

	server := &Server{
		name:           name,
		cfg:            cfg,
		scheduler:      scheduler,
		sessionManager: newSessionManager(scheduler, statsHandler),
		statsHandler:   statsHandler,
		stop:           make(chan bool),
		sessionTimeout: utils.ParseDurationOrDefault(*cfg.UdpSessionTimeout, 0),
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

	log := logging.For("UDPServer.Listen")

	this.statsHandler.Start()
	this.scheduler.Start()
	this.sessionManager.start()

	go func() {
		for {
			select {
			case <-this.stop:
				this.stopped = true
				this.sessionManager.Stop()
				this.scheduler.Stop()
				this.statsHandler.Stop()
				this.serverConn.Close()
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
func (this *Server) Listen() error {

	log := logging.For("UDPServer.Listen")

	listenAddr, err := net.ResolveUDPAddr("udp", this.cfg.Bind)
	this.serverConn, err = net.ListenUDP("udp", listenAddr)

	if err != nil {
		log.Error("Error starting UDP server: ", err)
		return err
	}

	// Main proxy loop goroutine
	go func() {
		for {
			buf := make([]byte, UDP_PACKET_SIZE)
			n, clientAddr, err := this.serverConn.ReadFromUDP(buf)

			if err != nil {
				if this.stopped {
					return
				}
				log.Error("Error ReadFromUDP: ", err)
				continue
			}

			go func(received []byte) {

				err := this.sessionManager.Send(this.serverConn, clientAddr, this.sessionTimeout, this.cfg.UdpResponses, received)
				if err != nil {
					log.Error("Error send to backend", err)
					return
				}

			}(buf[0:n])
		}
	}()

	return nil
}

/**
 * Stop, dropping all connections
 */
func (this *Server) Stop() {
	log := logging.For("server.Listen")
	log.Info("Stopping ", this.name)
	this.stop <- true
}
