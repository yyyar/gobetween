/**
 * server.go - proxy server implementation
 *
 * @author Yaroslav Pogrebnyak <yyyaroslav@gmail.com>
 */

package server

import (
	"../balance"
	"../config"
	"../core"
	"../discovery"
	"../healthcheck"
	"../logging"
	"../utils"
	"net"
)

/**
 * Server listens for client connections and
 * proxies it to backends
 */
type Server struct {

	/* Server friendly name */
	name string

	/* Listener */
	listener net.Listener

	/* Configuration */
	cfg config.Server

	/* Scheduler deals with discovery, balancing and healthchecks */
	scheduler Scheduler

	/* Current clients connection */
	clients map[string]net.Conn

	/* ----- channels ----- */

	/* Channel for new connections */
	connect chan (net.Conn)

	/* Channel for dropping connections or connectons to drop */
	disconnect chan (net.Conn)

	/* Stop channel */
	stop chan bool
}

/**
 * Creates new server instance
 */
func New(name string, cfg config.Server) *Server {

	log := logging.For("server")

	// Create server
	server := &Server{
		name:       name,
		cfg:        cfg,
		stop:       make(chan bool),
		disconnect: make(chan net.Conn),
		connect:    make(chan net.Conn),
		clients:    make(map[string]net.Conn),
		scheduler: Scheduler{
			balancer:    balance.New(cfg.Balance),
			discovery:   discovery.New(cfg.Discovery.Kind, *cfg.Discovery),
			healthcheck: healthcheck.New(cfg.Healthcheck.Kind, *cfg.Healthcheck),
		},
	}

	log.Info("Creating '", name, "': ", cfg.Bind, " ", cfg.Balance, " ", cfg.Discovery.Kind, " ", cfg.Healthcheck.Kind)

	return server
}

func (this *Server) Cfg() config.Server {
	return this.cfg
}

/**
 * Start server
 */
func (this *Server) Start() error {

	go func() {

		for {
			select {
			case client := <-this.disconnect:
				this.HandleClientDisconnect(client)

			case client := <-this.connect:
				this.HandleClientConnect(client)

			case <-this.stop:
				this.scheduler.Stop()
				if this.listener != nil {
					this.listener.Close()
					for _, conn := range this.clients {
						conn.Close()
					}
				}
				this.clients = make(map[string]net.Conn)
				return
			}
		}
	}()

	// Start scheduler
	this.scheduler.start()

	// Start listening
	if err := this.Listen(); err != nil {
		this.Stop()
		return err
	}

	return nil
}

/**
 * Handle client disconnection
 */
func (this *Server) HandleClientDisconnect(client net.Conn) {
	client.Close()
	delete(this.clients, client.RemoteAddr().String())
}

/**
 * Handle new client connection
 */
func (this *Server) HandleClientConnect(client net.Conn) {

	log := logging.For("server")

	if *this.cfg.MaxConnections != 0 && len(this.clients) >= *this.cfg.MaxConnections {
		log.Warn("Too many connections to ", this.cfg.Bind)
		client.Close()
		return
	}

	this.clients[client.RemoteAddr().String()] = client
	go func() {
		this.handle(client)
		this.disconnect <- client
	}()
}

/**
 * Stop, dropping all connections
 */
func (this *Server) Stop() {

	log := logging.For("server.Listen")
	log.Info("Stopping ", this.name)

	this.stop <- true
}

/**
 * Listen on specified port for a connections
 */
func (this *Server) Listen() (err error) {

	log := logging.For("server.Listen")

	if this.listener, err = net.Listen("tcp", this.cfg.Bind); err != nil {
		log.Error("Error starting TCP server: ", err)
		return err
	}

	go func() {
		for {
			conn, err := this.listener.Accept()
			if err != nil {
				log.Error(err)
				return
			}

			this.connect <- conn
		}
	}()

	return nil
}

/**
 * Handle incoming connection and prox it to backend
 */
func (this *Server) handle(clientConn net.Conn) {

	log := logging.For("server.handle")

	log.Debug("Accepted ", clientConn.RemoteAddr(), " -> ", this.listener.Addr())

	/* Find out backend for proxying */
	var err error
	backend, err := this.scheduler.TakeBackend(&core.Context{clientConn})
	if err != nil {
		log.Error(err, " Closing connection ", clientConn.RemoteAddr())
		return
	}

	/* Connect to backend */
	backendConn, err := net.DialTimeout(this.cfg.Protocol, backend.Address(), utils.ParseDurationOrDefault(*this.cfg.BackendConnectionTimeout, 0))
	if err != nil {
		log.Error(err)
		return
	}
	this.scheduler.IncrementConnection(*backend)
	defer this.scheduler.DecrementConnection(*backend)

	/* Stat proxying */
	log.Debug("Begin ", clientConn.RemoteAddr(), " -> ", this.listener.Addr(), " -> ", backendConn.RemoteAddr())
	clientStatsChan := proxy(clientConn, backendConn, utils.ParseDurationOrDefault(*this.cfg.ClientIdleTimeout, 0))
	backendStatsChan := proxy(backendConn, clientConn, utils.ParseDurationOrDefault(*this.cfg.BackendIdleTimeout, 0))

	/* Wait proxies to finish */
	writtenClient := <-clientStatsChan
	writtenBackend := <-backendStatsChan

	log.Info("Written to client: ", writtenClient, ", backend: ", writtenBackend)
	log.Debug("End ", clientConn.RemoteAddr(), " -> ", this.listener.Addr(), " -> ", backendConn.RemoteAddr())
}
