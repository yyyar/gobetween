/**
 * config.go - config file definitions
 *
 * @author Yaroslav Pogrebnyak <yyyaroslav@gmail.com>
 */
package config

import (
	"time"
)

/**
 * Config file top-level object
 */
type Config struct {
	Logging  LoggingConfig     `toml:"logging"`
	Defaults ConnectionOptions `toml:"defaults"`
	Servers  map[string]Server `toml:"servers"`
}

/**
 * Logging config section
 */
type LoggingConfig struct {
	Level  string `toml:"level"`
	Output string `toml:"output"`
}

/**
 * Default values can be overriden in server
 */
type ConnectionOptions struct {
	MaxConnections           *int        `toml:"max_connections"`
	ClientIdleTimeout        *MyDuration `toml:"client_idle_timeout"`
	BackendIdleTimeout       *MyDuration `toml:"backend_idle_timeout"`
	BackendConnectionTimeout *MyDuration `toml:"backend_connection_timeout"`
}

/**
 * Server section config
 */
type Server struct {
	ConnectionOptions

	// hostname:port
	Bind string `toml:"bind"`

	// tcp | udp
	Protocol string `toml:"protocol"`

	// weight | leastconn | roundrobin
	Balance string `toml:"balance"`

	// Discovery configuration
	Discovery *DiscoveryConfig `toml:"discovery"`

	// Healthcheck configuration
	Healthcheck *HealthcheckConfig `toml:"healthcheck"`
}

/**
 * Discovery configuration
 */
type DiscoveryConfig struct {
	Kind       string `toml:"kind"`
	Failpolicy string `toml:"failpolicy"`
	Interval   string `toml:"interval"`
	Timeout    string `toml:"timeout"`

	/* only if kind = "static" */
	StaticList []string `toml:"static_list"`

	/* only if kind = "srv" */
	SrvLookupServer  string `toml:"srv_lookup_server"`
	SrvLookupPattern string `toml:"srv_lookup_pattern"`

	/* only if kind = "docker" */
	DockerEndpoint             string `toml:"docker_endpoint"`
	DockerContainerLabel       string `toml:"docker_container_label"`
	DockerContainerPrivatePort int64  `toml:"docker_container_private_port"`

	/* only if kind = "json" */
	JsonEndpoint        string `toml:"json_endpoint"`
	JsonHostPattern     string `toml:"json_host_pattern"`
	JsonPortPattern     string `toml:"json_port_pattern"`
	JsonWeightPattern   string `toml:"json_weight_pattern"`
	JsonPriorityPattern string `toml:"json_priority_pattern"`

	/* only if kind = "exec" */
	ExecCommand []string `toml:"exec_command"`

	/* only if kind = "plaintext" */
	PlaintextEndpoint      string `toml:"plaintext_endpoint"`
	PlaintextRegexpPattern string `toml:"plaintext_regex_pattern"`
}

/**
 * Healthcheck configuration
 */
type HealthcheckConfig struct {
	Kind     string `toml:"kind"`
	Interval string `toml:"interval"`
	Passes   int    `toml:"passes"`
	Fails    int    `toml:"fails"`
	Timeout  string `toml:"timeout"`

	/* only if kind = "ping" */
	// nothing

	/* only if kind = "exec" */
	ExecCommand                string `toml:"exec_command"`
	ExecExpectedPositiveOutput string `toml:"exec_expected_positive_output"`
	ExecExpectedNegativeOutput string `toml:"exec_expected_negative_output"`
}

/* ----- Custom ----- */

/**
 * Custom duration struct for unmarshalling
 */
type MyDuration struct {
	Duration time.Duration
}

/**
 * Unmarshal duration fields
 */
func (d *MyDuration) UnmarshalText(text []byte) error {
	var err error
	s := string(text)
	if s == "" {
		d.Duration = 0
		return nil
	}
	d.Duration, err = time.ParseDuration(s)
	return err
}
