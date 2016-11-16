/**
 * server.go - UDP server implementation
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
	"../modules/access"
	"../scheduler"
	"errors"
	"net"
	"sync"
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

	/* collection of virtual udp sessions */
	sessions map[string]*session

	/* sessions modification mutex */
	sessionsLock sync.RWMutex

	/* Scheduler */
	scheduler *scheduler.Scheduler

	/* Stats handler */
	statsHandler *stats.Handler

	/* Server connection */
	serverConn *net.UDPConn

	/* Flag indicating that server is stopped */
	stopped bool

	/* Sessions will notify that they're closed to this channel */
	notify chan net.UDPAddr

	/* ----- modules ----- */

	/* Access module checks if client is allowed to connect */
	access *access.Access
}

/**
 * Creates new UDP server
 */
func New(name string, cfg config.Server) (*Server, error) {

	log := logging.For("udp/server")

	statsHandler := stats.NewHandler(name)
	scheduler := &scheduler.Scheduler{
		Balancer:     balance.New(cfg.Balance),
		Discovery:    discovery.New(cfg.Discovery.Kind, *cfg.Discovery),
		Healthcheck:  healthcheck.New(cfg.Healthcheck.Kind, *cfg.Healthcheck),
		StatsHandler: statsHandler,
	}

	server := &Server{
		name:         name,
		cfg:          cfg,
		scheduler:    scheduler,
		statsHandler: statsHandler,
		sessions:     make(map[string]*session),
		notify:       make(chan net.UDPAddr),
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
		for {
			clientAddr, more := <-this.notify
			if !more {
				return
			}
			this.removeSession(clientAddr)
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
				session := this.getSession(*clientAddr)

				if session == nil {
					var err error
					session, err = this.createSession(*clientAddr, this.serverConn)

					if err != nil {
						log.Error("Error creating session", err)
						return
					}
				}
				err := session.send(received)

				if err != nil {
					log.Error("Error sending data to backend", err)
				}

			}(buf[0:n])
		}
	}()

	return nil
}

/**
 * Creates new sessions; adds to itself and returns it
 */
func (this *Server) createSession(clientAddr net.UDPAddr, serverConn *net.UDPConn) (*session, error) {

	log := logging.For("udp/server")
	/* Check access if needed */
	if this.access != nil {
		if !this.access.Allows(&clientAddr.IP) {
			log.Debug("Client disallowed to connect ", clientAddr)
			return nil, errors.New("Access denied")
		}
	}

	log.Debug("Accepted ", clientAddr, " -> ", serverConn.LocalAddr())
	udpResponses := 0
	if this.cfg.Udp != nil {
		udpResponses = this.cfg.Udp.MaxResponses
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
		udpResponses:       udpResponses,
		scheduler:          this.scheduler,
		notify:             this.notify,
		serverConn:         serverConn,
		clientAddr:         clientAddr,
		backend:            backend,
	}

	err = session.start()
	if err != nil {
		session.stop()
		return nil, err
	}

	this.addSession(clientAddr, session)

	return session, nil
}

func (this *Server) addSession(clientAddr net.UDPAddr, session *session) {
	this.sessionsLock.Lock()
	this.sessions[clientAddr.String()] = session
	this.sessionsLock.Unlock()
}

func (this *Server) getSession(clientAddr net.UDPAddr) *session {
	this.sessionsLock.RLock()
	defer this.sessionsLock.RUnlock()
	return this.sessions[clientAddr.String()]
}

func (this *Server) removeSession(clientAddr net.UDPAddr) {
	this.sessionsLock.Lock()
	defer this.sessionsLock.Unlock()
	delete(this.sessions, clientAddr.String())
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

	this.sessionsLock.Lock()
	for _, session := range this.sessions {
		session.stop()
	}
	this.sessions = make(map[string]*session)
	this.sessionsLock.Unlock()

	close(this.notify)
}
