/**
 * config.go - config file definitions
 *
 * @author Yaroslav Pogrebnyak <yyyaroslav@gmail.com>
 */
package config

/**
 * Config file top-level object
 */
type Config struct {
	Logging  LoggingConfig     `toml:"logging"`
	Api      ApiConfig         `toml:"api"`
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
 * Api config section
 */
type ApiConfig struct {
	Enabled bool   `toml:"enabled"`
	Bind    string `toml:"bind"`
}

/**
 * Default values can be overridden in server
 */
type ConnectionOptions struct {
	MaxConnections           *int    `toml:"max_connections" json:"max_connections"`
	ClientIdleTimeout        *string `toml:"client_idle_timeout" json:"client_idle_timeout"`
	BackendIdleTimeout       *string `toml:"backend_idle_timeout" json:"backend_idle_timeout"`
	BackendConnectionTimeout *string `toml:"backend_connection_timeout" json:"backend_connection_timeout"`
}

/**
 * Server section config
 */
type Server struct {
	ConnectionOptions

	// hostname:port
	Bind string `toml:"bind" json:"bind"`

	// tcp | udp
	Protocol string `toml:"protocol" json:"protocol"`

	// weight | leastconn | roundrobin
	Balance string `toml:"balance" json:"balance"`

	// Discovery configuration
	Discovery *DiscoveryConfig `toml:"discovery" json:"discovery"`

	// Healthcheck configuration
	Healthcheck *HealthcheckConfig `toml:"healthcheck" json:"healthcheck"`
}

/**
 * Discovery configuration
 */
type DiscoveryConfig struct {
	Kind       string `toml:"kind" json:"kind"`
	Failpolicy string `toml:"failpolicy" json:"failpolicy"`
	Interval   string `toml:"interval" json:"interval"`
	Timeout    string `toml:"timeout" json:"timeout"`

	/* Depends on Kind */

	*StaticDiscoveryConfig
	*SrvDiscoveryConfig
	*DockerDiscoveryConfig
	*JsonDiscoveryConfig
	*ExecDiscoveryConfig
	*PlaintextDiscoveryConfig
}

type StaticDiscoveryConfig struct {
	StaticList []string `toml:"static_list" json:"static_list"`
}

type SrvDiscoveryConfig struct {
	SrvLookupServer  string `toml:"srv_lookup_server" json:"srv_lookup_server"`
	SrvLookupPattern string `toml:"srv_lookup_pattern" json:"srv_lookup_pattern"`
}

type ExecDiscoveryConfig struct {
	ExecCommand []string `toml:"exec_command" json:"exec_command"`
}

type JsonDiscoveryConfig struct {
	JsonEndpoint        string `toml:"json_endpoint" json:"json_endpoint"`
	JsonHostPattern     string `toml:"json_host_pattern" json:"json_host_pattern"`
	JsonPortPattern     string `toml:"json_port_pattern" json:"json_port_pattern"`
	JsonWeightPattern   string `toml:"json_weight_pattern" json:"json_weight_pattern"`
	JsonPriorityPattern string `toml:"json_priority_pattern" json:"json_priority_pattern"`
}

type PlaintextDiscoveryConfig struct {
	PlaintextEndpoint      string `toml:"plaintext_endpoint" json:"plaintext_endpoint"`
	PlaintextRegexpPattern string `toml:"plaintext_regex_pattern" json:"plaintext_regex_pattern"`
}

type DockerDiscoveryConfig struct {
	DockerEndpoint             string `toml:"docker_endpoint" json:"docker_endpoint"`
	DockerContainerLabel       string `toml:"docker_container_label" json:"docker_container_label"`
	DockerContainerPrivatePort int64  `toml:"docker_container_private_port" json:"docker_container_private_port"`
}

/**
 * Healthcheck configuration
 */
type HealthcheckConfig struct {
	Kind     string `toml:"kind" json:"kind"`
	Interval string `toml:"interval" json:"interval"`
	Passes   int    `toml:"passes" json:"passes"`
	Fails    int    `toml:"fails" json:"fails"`
	Timeout  string `toml:"timeout" json:"timeout"`

	/* Depends on Kind */

	*PingHealthcheckConfig
	*ExecHealthcheckConfig
}

type PingHealthcheckConfig struct{}

type ExecHealthcheckConfig struct {
	ExecCommand                string `toml:"exec_command" json:"exec_command,omitempty"`
	ExecExpectedPositiveOutput string `toml:"exec_expected_positive_output" json:"exec_expected_positive_output"`
	ExecExpectedNegativeOutput string `toml:"exec_expected_negative_output" json:"exec_expected_negative_output"`
}
