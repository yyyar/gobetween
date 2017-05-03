/**
 * server.go - UDP server implementation
 *
 * @author Illarion Kovalchuk <illarion.kovalchuk@gmail.com>
 */

package udp

import (
	"errors"
	"net"

	"../../balance"
	"../../config"
	"../../core"
	"../../discovery"
	"../../healthcheck"
	"../../logging"
	"../../stats"
	"../../utils"
	"../modules/access"
	"../scheduler"
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

	/* Stats handler */
	statsHandler *stats.Handler

	/* Server connection */
	serverConn *net.UDPConn

	/* Flag indicating that server is stopped */
	stopped bool

	/* ----- channels ----- */
	getOrCreate chan *sessionRequest
	remove      chan net.UDPAddr
	stop        chan bool

	/* ----- modules ----- */

	/* Access module checks if client is allowed to connect */
	access *access.Access
}

/**
 * Request to get session for clientAddr
 */
type sessionRequest struct {
	clientAddr net.UDPAddr
	response   chan sessionResponse
}

/**
 * Sessnion request response
 */
type sessionResponse struct {
	session *session
	err     error
}

/**
 * Creates new UDP server
 */
func New(name string, cfg config.Server) (*Server, error) {

	log := logging.For("udp/server")

	statsHandler := stats.NewHandler(name)
	scheduler := &scheduler.Scheduler{
		Balancer:     balance.New(nil, cfg.Balance),
		Discovery:    discovery.New(cfg.Discovery.Kind, *cfg.Discovery),
		Healthcheck:  healthcheck.New(cfg.Healthcheck.Kind, *cfg.Healthcheck),
		StatsHandler: statsHandler,
	}

	server := &Server{
		name:         name,
		cfg:          cfg,
		scheduler:    scheduler,
		statsHandler: statsHandler,
		getOrCreate:  make(chan *sessionRequest),
		remove:       make(chan net.UDPAddr),
		stop:         make(chan bool),
	}

	/* Add access if needed */
	if cfg.Access != nil {
		access, err := access.NewAccess(cfg.Access)
		if err != nil {
			return nil, err
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

	log := logging.For("udp/server")

	this.statsHandler.Start()
	this.scheduler.Start()

	// Start listening
	if err := this.listen(); err != nil {
		this.Stop()
		log.Error("Error starting UDP Listen ", err)
		return err
	}

	go func() {
		sessions := make(map[string]*session)
		for {
			select {

			/* handle get session request */
			case sessionRequest := <-this.getOrCreate:
				session, ok := sessions[sessionRequest.clientAddr.String()]

				if ok {
					sessionRequest.response <- sessionResponse{
						session: session,
						err:     nil,
					}
					break
				}

				session, err := this.makeSession(sessionRequest.clientAddr)
				if err == nil {
					sessions[sessionRequest.clientAddr.String()] = session
				}

				sessionRequest.response <- sessionResponse{
					session: session,
					err:     err,
				}

			/* handle session remove */
			case clientAddr := <-this.remove:
				session, ok := sessions[clientAddr.String()]
				if !ok {
					break
				}
				session.stop()
				delete(sessions, clientAddr.String())

			/* handle server stop */
			case <-this.stop:
				for _, session := range sessions {
					session.stop()
				}
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

	log := logging.For("udp/server")

	listenAddr, err := net.ResolveUDPAddr("udp", this.cfg.Bind)
	if err != nil {
		log.Error("Error resolving server bind addr ", err)
		return err
	}

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

			go func(buf []byte) {
				responseChan := make(chan sessionResponse, 1)

				this.getOrCreate <- &sessionRequest{
					clientAddr: *clientAddr,
					response:   responseChan,
				}

				response := <-responseChan

				if response.err != nil {
					log.Error("Error creating session ", response.err)
					return
				}

				err := response.session.send(buf)

				if err != nil {
					log.Error("Error sending data to backend ", err)
				}

			}(buf[0:n])
		}
	}()

	return nil
}

/**
 * Makes new session
 */
func (this *Server) makeSession(clientAddr net.UDPAddr) (*session, error) {

	log := logging.For("udp/server")
	/* Check access if needed */
	if this.access != nil {
		if !this.access.Allows(&clientAddr.IP) {
			log.Debug("Client disallowed to connect ", clientAddr)
			return nil, errors.New("Access denied")
		}
	}

	log.Debug("Accepted ", clientAddr, " -> ", this.serverConn.LocalAddr())

	var maxRequests uint64
	var maxResponses uint64

	if this.cfg.Udp != nil {
		maxRequests = this.cfg.Udp.MaxRequests
		maxResponses = this.cfg.Udp.MaxResponses
	}

	backend, err := this.scheduler.TakeBackend(&core.UdpContext{
		RemoteAddr: clientAddr,
	})

	if err != nil {
		return nil, err
	}

	session := &session{
		clientIdleTimeout:  utils.ParseDurationOrDefault(*this.cfg.ClientIdleTimeout, 0),
		backendIdleTimeout: utils.ParseDurationOrDefault(*this.cfg.BackendIdleTimeout, 0),
		maxRequests:        maxRequests,
		maxResponses:       maxResponses,
		scheduler:          this.scheduler,
		notifyClosed: func() {
			this.remove <- clientAddr
		},
		serverConn: this.serverConn,
		clientAddr: clientAddr,
		backend:    backend,
	}

	err = session.start()
	if err != nil {
		session.stop()
		return nil, err
	}

	return session, nil
}

/**
 * Stop, dropping all connections
 */
func (this *Server) Stop() {
	log := logging.For("udp/server")
	log.Info("Stopping ", this.name)

	this.stopped = true
	this.serverConn.Close()

	this.scheduler.Stop()
	this.statsHandler.Stop()
	this.stop <- true
}
