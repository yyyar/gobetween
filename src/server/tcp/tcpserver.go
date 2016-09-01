/**
 * tcpserver.go - TCP server implementation
 *
 * @author Yaroslav Pogrebnyak <yyyaroslav@gmail.com>
 */
package tcp

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
	"net"
)

/**
 * Server listens for client connections and
 * proxies it to backends
 */
type TCPServer struct {

	/* Server friendly name */
	name string

	/* Listener */
	listener net.Listener

	/* Configuration */
	cfg config.Server

	/* Scheduler deals with discovery, balancing and healthchecks */
	scheduler scheduler.Scheduler

	/* Current clients connection */
	clients map[string]net.Conn

	/* Stats handler */
	statsHandler *stats.Handler

	/* ----- channels ----- */

	/* Channel for new connections */
	connect chan (net.Conn)

	/* Channel for dropping connections or connectons to drop */
	disconnect chan (net.Conn)

	/* Stop channel */
	stop chan bool

	/* ----- modules ----- */

	/* Access module checks if client is allowed to connect */
	access *access.Access
}

/**
 * Creates new server instance
 */
func NewTCPServer(name string, cfg config.Server) (*TCPServer, error) {

	log := logging.For("server")

	var err error = nil

	// Create server
	server := &TCPServer{
		name:         name,
		cfg:          cfg,
		stop:         make(chan bool),
		disconnect:   make(chan net.Conn),
		connect:      make(chan net.Conn),
		clients:      make(map[string]net.Conn),
		statsHandler: stats.NewHandler(name),
		scheduler: scheduler.Scheduler{
			Balancer:    balance.New(cfg.Balance),
			Discovery:   discovery.New(cfg.Discovery.Kind, *cfg.Discovery),
			Healthcheck: healthcheck.New(cfg.Healthcheck.Kind, *cfg.Healthcheck),
		},
	}

	/* Add access if needed */
	if cfg.Access != nil {
		server.access, err = access.NewAccess(cfg.Access)
		if err != nil {
			return nil, err
		}
	}

	// TODO: refatror
	server.scheduler.StatsHandler = server.statsHandler

	log.Info("Creating '", name, "': ", cfg.Bind, " ", cfg.Balance, " ", cfg.Discovery.Kind, " ", cfg.Healthcheck.Kind)

	return server, nil
}

/**
 * Returns current server configuration
 */
func (this *TCPServer) Cfg() config.Server {
	return this.cfg
}

/**
 * Start server
 */
func (this *TCPServer) Start() error {

	go func() {

		for {
			select {
			case client := <-this.disconnect:
				this.HandleClientDisconnect(client)

			case client := <-this.connect:
				this.HandleClientConnect(client)

			case <-this.stop:
				this.scheduler.Stop()
				this.statsHandler.Stop()
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

	// Start stats handler
	this.statsHandler.Start()

	// Start scheduler
	this.scheduler.Start()

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
func (this *TCPServer) HandleClientDisconnect(client net.Conn) {
	client.Close()
	delete(this.clients, client.RemoteAddr().String())
	this.statsHandler.Connections <- uint(len(this.clients))
}

/**
 * Handle new client connection
 */
func (this *TCPServer) HandleClientConnect(client net.Conn) {

	log := logging.For("server")

	if *this.cfg.MaxConnections != 0 && len(this.clients) >= *this.cfg.MaxConnections {
		log.Warn("Too many connections to ", this.cfg.Bind)
		client.Close()
		return
	}

	this.clients[client.RemoteAddr().String()] = client
	this.statsHandler.Connections <- uint(len(this.clients))
	go func() {
		this.handle(client)
		this.disconnect <- client
	}()
}

/**
 * Stop, dropping all connections
 */
func (this *TCPServer) Stop() {

	log := logging.For("server.Listen")
	log.Info("Stopping ", this.name)

	this.stop <- true
}

/**
 * Listen on specified port for a connections
 */
func (this *TCPServer) Listen() (err error) {

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
func (this *TCPServer) handle(clientConn net.Conn) {

	log := logging.For("server.handle")

	/* Check access if needed */
	if this.access != nil {
		if !this.access.Allows(&clientConn.RemoteAddr().(*net.TCPAddr).IP) {
			log.Debug("Client disallowed to connect ", clientConn.RemoteAddr())
			clientConn.Close()
			return
		}
	}

	log.Debug("Accepted ", clientConn.RemoteAddr(), " -> ", this.listener.Addr())

	/* Find out backend for proxying */
	var err error
	backend, err := this.scheduler.TakeBackend(&core.TcpContext{clientConn})
	if err != nil {
		log.Error(err, " Closing connection ", clientConn.RemoteAddr())
		return
	}

	/* Connect to backend */
	backendConn, err := net.DialTimeout(this.cfg.Protocol, backend.Address(), utils.ParseDurationOrDefault(*this.cfg.BackendConnectionTimeout, 0))
	if err != nil {
		this.scheduler.IncrementRefused(*backend)
		log.Error(err)
		return
	}
	this.scheduler.IncrementConnection(*backend)
	defer this.scheduler.DecrementConnection(*backend)

	/* Stat proxying */
	log.Debug("Begin ", clientConn.RemoteAddr(), " -> ", this.listener.Addr(), " -> ", backendConn.RemoteAddr())
	cs := proxy(clientConn, backendConn, utils.ParseDurationOrDefault(*this.cfg.ClientIdleTimeout, 0))
	bs := proxy(backendConn, clientConn, utils.ParseDurationOrDefault(*this.cfg.BackendIdleTimeout, 0))

	isTx, isRx := true, true
	for isTx || isRx {
		select {
		case s, ok := <-cs:
			isRx = ok
			this.scheduler.IncrementRx(*backend, s.CountWrite)
		case s, ok := <-bs:
			isTx = ok
			this.scheduler.IncrementTx(*backend, s.CountWrite)
		}
	}
}
