package manager

/**
 * manager.go - manages servers
 *
 * @author Yaroslav Pogrebnyak <yyyaroslav@gmail.com>
 */

import (
	"errors"
	"os"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/yyyar/gobetween/config"
	"github.com/yyyar/gobetween/core"
	"github.com/yyyar/gobetween/logging"
	"github.com/yyyar/gobetween/server"
	"github.com/yyyar/gobetween/service"
	"github.com/yyyar/gobetween/utils/codec"
	"github.com/yyyar/gobetween/utils/profiler"
)

/* Map of app current servers */
var servers = struct {
	sync.RWMutex
	m map[string]core.Server
}{m: make(map[string]core.Server)}

/* default configuration for server */
var defaults config.ConnectionOptions

/* services */
var services []core.Service

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
	initDefaults()

	//Initialize global sections
	initConfigGlobals(&cfg)

	//create services
	services = service.All(cfg)

	// Go through config and start servers for each server
	for name, serverCfg := range cfg.Servers {
		err := Create(name, serverCfg)
		if err != nil {
			log.Fatal(err)
		}
	}

	// Initialize profiler
	initProfiler(&cfg)

	log.Info("Initialized")
}

func initDefaults() {
	//defaults
	if defaults.MaxConnections == nil {
		defaults.MaxConnections = new(int)
	}

	if defaults.ClientIdleTimeout == nil {
		defaults.ClientIdleTimeout = new(string)
		*defaults.ClientIdleTimeout = "0"
	}

	if defaults.BackendIdleTimeout == nil {
		defaults.BackendIdleTimeout = new(string)
		*defaults.BackendIdleTimeout = "0"
	}

	if defaults.BackendConnectionTimeout == nil {
		defaults.BackendConnectionTimeout = new(string)
		*defaults.BackendConnectionTimeout = "0"
	}
}

func initConfigGlobals(cfg *config.Config) {

	//acme
	if cfg.Acme != nil {
		if cfg.Acme.Challenge == "" {
			cfg.Acme.Challenge = "http"
		}

		if cfg.Acme.HttpBind == "" {
			cfg.Acme.HttpBind = "0.0.0.0:80"
		}

		if cfg.Acme.CacheDir == "" {
			cfg.Acme.CacheDir = "/tmp"
		}
	}
}

func initProfiler(cfg *config.Config) {
	if cfg.Profiler == nil {
		return
	}

	if !cfg.Profiler.Enabled {
		return
	}

	profiler.Start(cfg.Profiler.Bind)
}

/**
 * Dumps current [servers] section to
 * the config file
 */
func DumpConfig(format string) (string, error) {

	originalCfg.Servers = map[string]config.Server{}

	servers.RLock()
	for name, server := range servers.m {
		originalCfg.Servers[name] = server.Cfg()
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
	for name, server := range servers.m {
		result[name] = server.Cfg()
	}
	servers.RUnlock()

	return result
}

/**
 * Returns server configuration by name
 */
func Get(name string) interface{} {

	servers.RLock()
	server, ok := servers.m[name]
	servers.RUnlock()

	if !ok {
		return nil
	}

	return server.Cfg()
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

	server, err := server.New(name, c)

	if err != nil {
		return err
	}

	for _, srv := range services {
		err = srv.Enable(server)
		if err != nil {
			return err
		}
	}

	if err = server.Start(); err != nil {
		return err
	}

	servers.m[name] = server

	return nil
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

	for _, s := range services {
		s.Disable(server)
	}

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
		"probe",
		"exec",
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

	if server.Healthcheck.Kind != "none" {
		d, err := time.ParseDuration(server.Healthcheck.Interval)
		if err != nil {
			return config.Server{}, errors.New("Could not parse healtcheck interval: " + err.Error())
		}

		if d <= 0 {
			return config.Server{}, errors.New("Healthcheck interval should be greater than 0s")
		}
	}

	if server.Healthcheck.Kind == "probe" {

		switch server.Healthcheck.ProbeProtocol {
		case "tcp", "udp":
		default:
			return config.Server{}, errors.New("Unsupported probe_protocol")
		}

		if server.Healthcheck.ProbeSend == "" || server.Healthcheck.ProbeRecv == "" {
			return config.Server{}, errors.New("probe healthcheck should have both probe_send and probe_recv specified")
		}

		if server.Healthcheck.ProbeStrategy == "" {
			server.Healthcheck.ProbeStrategy = "starts_with"
		}

		var err error
		server.Healthcheck.ProbeSend, err = strconv.Unquote("\"" + server.Healthcheck.ProbeSend + "\"")
		if err != nil {
			return config.Server{}, errors.New("probe_send has invalid syntax " + err.Error())
		}

		switch server.Healthcheck.ProbeStrategy {
		case "starts_with":
			if server.Healthcheck.ProbeRecvLen > 0 {
				return config.Server{}, errors.New("probe_recv_len is redundant for 'starts_with' strategy")
			}

			var err error
			server.Healthcheck.ProbeRecv, err = strconv.Unquote("\"" + server.Healthcheck.ProbeRecv + "\"")
			if err != nil {
				return config.Server{}, errors.New("probe_recv has invalid syntax " + err.Error())
			}
		case "regexp":
			if server.Healthcheck.ProbeRecvLen == 0 {
				return config.Server{}, errors.New("probe_recv_len required")
			}

			_, err := regexp.Compile(server.Healthcheck.ProbeRecv)
			if err != nil {
				return config.Server{}, errors.New("probe_recv has invalid syntax " + err.Error())
			}
		default:
			return config.Server{}, errors.New("Unsupported probe_strategy " + server.Healthcheck.ProbeStrategy)
		}

	}

	if server.ProxyProtocol != nil {

		if server.Protocol != "tcp" {
			return config.Server{}, errors.New("proxy_protocol may be used only with 'tcp' protocol, not with " + server.Protocol)
		}

		if server.ProxyProtocol.Version == "" {
			return config.Server{}, errors.New("version field for proxy_protocol is not specified")
		}

		if server.ProxyProtocol.Version != "1" {
			return config.Server{}, errors.New("Unsupported proxy_protocol version " + server.ProxyProtocol.Version)
		}
	}

	if server.Sni != nil {

		if server.Sni.ReadTimeout == "" {
			server.Sni.ReadTimeout = "2s"
		}

		if server.Sni.UnexpectedHostnameStrategy == "" {
			server.Sni.UnexpectedHostnameStrategy = "default"
		}

		switch server.Sni.UnexpectedHostnameStrategy {
		case
			"default",
			"reject",
			"any":
		default:
			return config.Server{}, errors.New("Not supported sni unexprected hostname strategy " + server.Sni.UnexpectedHostnameStrategy)
		}

		if server.Sni.HostnameMatchingStrategy == "" {
			server.Sni.HostnameMatchingStrategy = "exact"
		}

		switch server.Sni.HostnameMatchingStrategy {
		case
			"exact",
			"regexp":
		default:
			return config.Server{}, errors.New("Not supported sni matching " + server.Sni.HostnameMatchingStrategy)
		}

		if _, err := time.ParseDuration(server.Sni.ReadTimeout); err != nil {
			return config.Server{}, errors.New("timeout parsing error")
		}
	}

	if _, err := time.ParseDuration(server.Healthcheck.Timeout); err != nil {
		return config.Server{}, errors.New("timeout parsing error")
	}

	if _, err := time.ParseDuration(server.Healthcheck.Interval); err != nil {
		return config.Server{}, errors.New("interval parsing error")
	}

	if server.BackendsTls != nil && ((server.BackendsTls.KeyPath == nil) != (server.BackendsTls.CertPath == nil)) {
		return config.Server{}, errors.New("backend_tls.cert_path and .key_path should be specified together")
	}

	if server.Tls != nil {

		if (len(server.Tls.AcmeHosts) == 0) && ((server.Tls.KeyPath == "") || (server.Tls.CertPath == "")) {
			return config.Server{}, errors.New("tls requires specify either acme hosts or both key and cert paths")
		}

	}

	/* ----- Connections params and overrides ----- */

	/* Protocol */
	switch server.Protocol {
	case "":
		server.Protocol = "tcp"
	case "tls":
		if server.Tls == nil {
			return config.Server{}, errors.New("Need tls section for tls protocol")
		}
		fallthrough
	case "tcp":
	case "udp":
		if server.BackendsTls != nil {
			return config.Server{}, errors.New("backends_tls should not be enabled for udp protocol")
		}

		if server.Udp == nil {
			server.Udp = &config.Udp{}
		}

		if server.Udp.MaxRequests == 0 && server.Udp.MaxResponses == 0 && server.ClientIdleTimeout == nil && server.BackendIdleTimeout == nil {
			return config.Server{}, errors.New("udp protocol requires to specify at least one of (client|backend)_idle_timeout, udp.max_requests, udp.max_responses")
		}

	default:
		return config.Server{}, errors.New("Not supported protocol " + server.Protocol)
	}

	/* Healthcheck and protocol match */

	if server.Healthcheck.Kind == "ping" && server.Protocol == "udp" {
		return config.Server{}, errors.New("Cant use ping healthcheck with udp server")
	}

	/* Balance */
	switch server.Balance {
	case
		"weight",
		"leastconn",
		"roundrobin",
		"leastbandwidth",
		"iphash1",
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
			server.Discovery.SrvDnsProtocol = "udp"
		default:
			return config.Server{}, errors.New("Not supported srv_dns_protocol " + server.Discovery.SrvDnsProtocol)
		}
	}

	/* LXD Discovery */
	if server.Discovery.Kind == "lxd" {

		if server.Discovery.LXDServerAddress == "" {
			return config.Server{}, errors.New("lxd_server_address is required" + server.Discovery.LXDServerAddress)
		}

		if !(strings.HasPrefix(server.Discovery.LXDServerAddress, "https:") ||
			strings.HasPrefix(server.Discovery.LXDServerAddress, "unix:")) {

			return config.Server{}, errors.New("lxd_server_address should start with either unix:// or https:// but got " + server.Discovery.LXDServerAddress)
		}

		if server.Discovery.LXDServerRemoteName == "" {
			server.Discovery.LXDServerRemoteName = "local"
		}

		if server.Discovery.LXDConfigDirectory == "" {
			server.Discovery.LXDConfigDirectory = os.ExpandEnv("$HOME/.config/lxc")
		}

		if server.Discovery.LXDContainerInterface == "" {
			server.Discovery.LXDContainerInterface = "eth0"
		}

		switch server.Discovery.LXDContainerAddressType {
		case
			"IPv4",
			"IPv6":
		case "":
			server.Discovery.LXDContainerAddressType = "IPv4"
		default:
			return config.Server{}, errors.New("Invalid lxd_container_address_type. Must be IPv4 or IPv6")
		}

	}

	/* TODO: Still need to decide how to get rid of this */

	if server.MaxConnections == nil {
		server.MaxConnections = new(int)
		*server.MaxConnections = *defaults.MaxConnections
	}

	if server.ClientIdleTimeout == nil {
		server.ClientIdleTimeout = new(string)
		*server.ClientIdleTimeout = *defaults.ClientIdleTimeout
	}

	if server.BackendIdleTimeout == nil {
		server.BackendIdleTimeout = new(string)
		*server.BackendIdleTimeout = *defaults.BackendIdleTimeout
	}

	if server.BackendConnectionTimeout == nil {
		server.BackendConnectionTimeout = new(string)
		*server.BackendConnectionTimeout = *defaults.BackendConnectionTimeout
	}

	return server, nil
}
