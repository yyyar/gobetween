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
	"../server/tcp"
	"../server/udp"
	"../utils/codec"
	"encoding/hex"
	"errors"
	"regexp"
	"strings"
	"sync"
	"time"
)

/* Map of app current servers */
var servers = struct {
	sync.RWMutex
	m map[string]server.Server
}{m: make(map[string]server.Server)}

/* default configuration for server */
var defaults config.ConnectionOptions

/* original cfg read from the file */
var originalCfg config.Config

/**
 * Initialize manager from the initial/default configuration
 */
func Initialize(cfg config.Config) {

	log := logging.For("manager")
	log.Info("Initializing...")

	originalCfg = cfg

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
 * Dumps current [servers] section to
 * the config file
 */
func DumpConfig(format string) (string, error) {

	originalCfg.Servers = map[string]config.Server{}

	servers.RLock()
	for name, srv := range servers.m {
		originalCfg.Servers[name] = srv.Cfg()
	}
	servers.RUnlock()

	var out *string = new(string)
	if err := codec.Encode(originalCfg, out, format); err != nil {
		return "", err
	}

	return *out, nil
}

/**
 * Returns map of servers with configurations
 */
func All() map[string]config.Server {
	result := map[string]config.Server{}

	servers.RLock()
	for name, srv := range servers.m {
		result[name] = srv.Cfg()
	}
	servers.RUnlock()

	return result
}

/**
 * Returns server configuration by name
 */
func Get(name string) interface{} {

	servers.RLock()
	srv, ok := servers.m[name]
	servers.RUnlock()

	if !ok {
		return nil
	}

	return srv.Cfg()
}

/**
 * Create new server and launch it
 */
func Create(name string, cfg config.Server) error {

	servers.Lock()
	defer servers.Unlock()

	if _, ok := servers.m[name]; ok {
		return errors.New("Server with this name already exists: " + name)
	}

	c, err := prepareConfig(name, cfg, defaults)
	if err != nil {
		return err
	}

	{
		var srv server.Server
		var err error

		switch c.Protocol {
		case "tcp":
			srv, err = tcp.NewTCPServer(name, c)
		case "udp":
			srv, err = udp.NewUDPServer(name, c)
		default:
			return errors.New("Unknown server type for protocol" + c.Protocol)
		}

		if err != nil {
			return err
		}
		servers.m[name] = srv

		return srv.Start()
	}
}

/**
 * Delete server stopping all active connections
 */
func Delete(name string) error {

	servers.Lock()
	defer servers.Unlock()

	server, ok := servers.m[name]
	if !ok {
		return errors.New("Server not found")
	}

	server.Stop()
	delete(servers.m, name)

	return nil
}

/**
 * Returns stats for the server
 */
func Stats(name string) interface{} {

	servers.Lock()
	server := servers.m[name]
	servers.Unlock()

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
		server.Healthcheck = &config.HealthcheckConfig{
			Kind:     "none",
			Interval: "0",
			Timeout:  "0",
		}
	}

	switch server.Healthcheck.Kind {
	case
		"ping",
		"exec",
		"udp",
		"none":
	default:
		return config.Server{}, errors.New("Not supported healthcheck type " + server.Healthcheck.Kind)
	}

	if server.Healthcheck.Interval == "" {
		server.Healthcheck.Interval = "0"
	}

	if server.Healthcheck.Timeout == "" {
		server.Healthcheck.Timeout = "0"
	}

	if server.Healthcheck.Fails <= 0 {
		server.Healthcheck.Fails = 1
	}

	if server.Healthcheck.Passes <= 0 {
		server.Healthcheck.Passes = 1
	}

	if _, err := time.ParseDuration(server.Healthcheck.Timeout); err != nil {
		return config.Server{}, errors.New("timeout parsing error")
	}

	if _, err := time.ParseDuration(server.Healthcheck.Interval); err != nil {
		return config.Server{}, errors.New("interval parsing error")
	}

	/* ----- Connections params and overrides ----- */

	/* Balance */
	switch server.Protocol {
	case
		"tcp",
		"udp":
	case "":
		server.Protocol = "tcp"
	default:
		return config.Server{}, errors.New("Not supported protocol " + server.Protocol)
	}

	/* Healthcheck and protocol match */

	if server.Healthcheck.Kind == "udp" && server.Protocol != "udp" {
		return config.Server{}, errors.New("Not supported healthcheck kind by server protocol")
	}

	if server.Healthcheck.Kind == "ping" && server.Protocol == "udp" {
		return config.Server{}, errors.New("Not supported healthcheck kind by server protocol")
	}

	/* UDP healthcheck */
	if server.Healthcheck.Kind == "udp" {

		if _, err := hex.DecodeString(strings.Replace(server.Healthcheck.SendPattern, " ", "", -1)); err != nil {
			return config.Server{}, errors.New("send_pattern parsing error")
		}

		if server.Healthcheck.ExpectedPattern != nil {
			pattern := strings.Replace(*server.Healthcheck.ExpectedPattern, " ", "", -1)

			_, err := regexp.Compile(pattern)

			if err != nil {
				return config.Server{}, errors.New("invalid regexp in expected_pattern")
			}
		}

	}

	/* Balance */
	switch server.Balance {
	case
		"weight",
		"leastconn",
		"roundrobin",
		"leastbandwidth",
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

	/* SRV Discovery */
	if server.Discovery.Kind == "srv" {
		switch server.Discovery.SrvDnsProtocol {
		case
			"udp",
			"tcp":
		case "":
			server.Discovery.Failpolicy = "udp"
		default:
			return config.Server{}, errors.New("Not supported srv_dns_protocol " + server.Discovery.SrvDnsProtocol)
		}
	}

	/* TODO: Still need to decide how to get rid of this */

	if defaults.MaxConnections == nil {
		defaults.MaxConnections = new(int)
	}
	if server.MaxConnections == nil {
		server.MaxConnections = defaults.MaxConnections
	}

	if defaults.ClientIdleTimeout == nil {
		defaults.ClientIdleTimeout = new(string)
		*defaults.ClientIdleTimeout = "0"
	}
	if server.ClientIdleTimeout == nil {
		server.ClientIdleTimeout = new(string)
		*server.ClientIdleTimeout = *defaults.ClientIdleTimeout
	}

	if defaults.BackendIdleTimeout == nil {
		defaults.BackendIdleTimeout = new(string)
		*defaults.BackendIdleTimeout = "0"
	}
	if server.BackendIdleTimeout == nil {
		server.BackendIdleTimeout = new(string)
		*server.BackendIdleTimeout = *defaults.BackendIdleTimeout
	}

	if defaults.BackendConnectionTimeout == nil {
		defaults.BackendConnectionTimeout = new(string)
		*defaults.BackendConnectionTimeout = "0"
	}
	if server.BackendConnectionTimeout == nil {
		server.BackendConnectionTimeout = new(string)
		*server.BackendConnectionTimeout = *defaults.BackendConnectionTimeout
	}

	return server, nil
}
