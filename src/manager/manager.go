/**
 * manager.go - manages servers
 *
 * @author Yaroslav Pogrebnyak <yyyaroslav@gmail.com>
 */
package manager

import (
	"../config"
	"../logging"
	"../server"
	"errors"
)

/* Map of app current servers */
var servers = map[string]*server.Server{}

/* default configuration for server */
var defaults config.ConnectionOptions

/**
 * Initialize manager from the initial/default configuration
 */
func Initialize(cfg config.Config) {

	log := logging.For("manager")
	log.Info("Initializing...")

	// save defaults for futher reuse
	defaults = cfg.Defaults

	// Go through config and start servers for each server
	for name, serverCfg := range cfg.Servers {
		err := Create(name, serverCfg)
		if err != nil {
			log.Fatal(err)
		}
	}

	log.Info("Initialized")
}

/**
 * Returns map of servers with configurations
 */
func All() map[string]config.Server {
	result := map[string]config.Server{}
	for name, server := range servers {
		result[name] = server.Cfg()
	}
	return result
}

/**
 * Returns server configuration by name
 */
func Get(name string) interface{} {

	server, ok := servers[name]
	if !ok {
		return nil
	}

	return server.Cfg()
}

/**
 * Create new server and launch it
 */
func Create(name string, cfg config.Server) error {
	c, err := prepareConfig(name, cfg, defaults)
	if err != nil {
		return err
	}
	server := server.New(name, c)
	servers[name] = server
	return server.Start()
}

/**
 * Delete server stopping all active connections
 */
func Delete(name string) error {

	server, ok := servers[name]
	if !ok {
		return errors.New("Server not found")
	}

	server.Stop()
	delete(servers, name)

	return nil
}

/**
 * Reconfigure existing server on-the-fly
 */
func Reconfigure(serverName string, cfg config.Server) {
	// TODO
}

/**
 * Returns stats for the server
 */
func Stats(name string) interface{} {
	server := servers[name]
	return server
}

/**
 * Prepare config (merge default configuration, and try to validate)
 * TODO: make validation better
 */
func prepareConfig(name string, server config.Server, defaults config.ConnectionOptions) (config.Server, error) {

	/* ----- Prerequisites ----- */

	if server.Bind == "" {
		return config.Server{}, errors.New("No bind specified")
	}

	if server.Discovery == nil {
		return config.Server{}, errors.New("No .discovery specified")
	}

	if server.Healthcheck == nil {
		return config.Server{}, errors.New("No .healthcheck specified. Will allow it later :-)")
	}

	/* ----- Connections params and overrides ----- */

	/* Balance */
	switch server.Protocol {
	case
		"tcp":
	case "":
		server.Protocol = "tcp"
	default:
		return config.Server{}, errors.New("Not supported protocol " + server.Protocol)
	}

	/* Balance */
	switch server.Balance {
	case
		"weight",
		"leastconn",
		"roundrobin",
		"iphash":
	case "":
		server.Balance = "weight"
	default:
		return config.Server{}, errors.New("Not supported balance type " + server.Balance)
	}

	/* Discovery */
	switch server.Discovery.Failpolicy {
	case
		"keeplast",
		"setempty":
	case "":
		server.Discovery.Failpolicy = "keeplast"
	default:
		return config.Server{}, errors.New("Not supported failpolicy " + server.Discovery.Failpolicy)
	}

	if server.Discovery.Interval == "" {
		server.Discovery.Interval = "0"
	}

	if server.Discovery.Timeout == "" {
		server.Discovery.Timeout = "0"
	}

	/* TODO: Still need to decide how to get rid of this */

	if defaults.MaxConnections == nil {
		defaults.MaxConnections = new(int)
	}
	if server.MaxConnections == nil {
		server.MaxConnections = defaults.MaxConnections
	}

	if defaults.ClientIdleTimeout == nil {
		*defaults.ClientIdleTimeout = "0"
	}
	if server.ClientIdleTimeout == nil {
		server.ClientIdleTimeout = new(string)
		*server.ClientIdleTimeout = *defaults.ClientIdleTimeout
	}

	if defaults.BackendIdleTimeout == nil {
		*defaults.BackendIdleTimeout = "0"
	}
	if server.BackendIdleTimeout == nil {
		server.BackendIdleTimeout = new(string)
		*server.BackendIdleTimeout = *defaults.BackendIdleTimeout
	}

	if defaults.BackendConnectionTimeout == nil {
		*defaults.BackendConnectionTimeout = "0"
	}
	if server.BackendConnectionTimeout == nil {
		server.BackendConnectionTimeout = new(string)
		*server.BackendConnectionTimeout = *defaults.BackendConnectionTimeout
	}

	return server, nil
}
