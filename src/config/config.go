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
	Logging  LoggingConfig     `toml:"logging" json:"logging"`
	Api      ApiConfig         `toml:"api" json:"api"`
	Defaults ConnectionOptions `toml:"defaults" json:"defaults"`
	Servers  map[string]Server `toml:"servers" json:"servers"`
}

/**
 * Logging config section
 */
type LoggingConfig struct {
	Level  string `toml:"level" json:"level"`
	Output string `toml:"output" json:"output"`
}

/**
 * Api config section
 */
type ApiConfig struct {
	Enabled   bool                `toml:"enabled" json:"enabled"`
	Bind      string              `toml:"bind" json:"bind"`
	BasicAuth *ApiBasicAuthConfig `toml:"basic_auth" json:"basic_auth"`
	Tls       *ApiTlsConfig       `toml:"tls" json:"tls"`
}

/**
 * Api Basic Auth Config
 */
type ApiBasicAuthConfig struct {
	Login    string `toml:"login" json:"login"`
	Password string `toml:"password" json:"password"`
}

/**
 * Api TLS server Config
 */
type ApiTlsConfig struct {
	CertPath string `toml:"cert_path" json:"cert_path"`
	KeyPath  string `toml:"key_path" json:"key_path"`
}

/**
 * Default values can be overridden in server
 */
type ConnectionOptions struct {
	MaxConnections           *int    `toml:"max_connections" json:"max_connections"`
	ClientIdleTimeout        *string `toml:"client_idle_timeout" json:"client_idle_timeout"`
	BackendIdleTimeout       *string `toml:"backend_idle_timeout" json:"backend_idle_timeout"`
	BackendConnectionTimeout *string `toml:"backend_connection_timeout" json:"backend_connection_timeout"`
	BackendTlsEnabled        *bool   `toml:"backend_tls_enabled" json:"backend_tls_enabled"`
	BackendTlsVerify         *bool   `toml:"backend_tls_verify" json:"backend_tls_verify"`
}

/**
 * Server section config
 */
type Server struct {
	ConnectionOptions

	// hostname:port
	Bind string `toml:"bind" json:"bind"`

	// tcp | udp | tls
	Protocol string `toml:"protocol" json:"protocol"`

	// weight | leastconn | roundrobin
	Balance string `toml:"balance" json:"balance"`

	// Optional configuration for protocol = tls
	Tls *Tls `toml:"tls" json:"tls"`

	// Optional configuration for backend_tls_enabled = true
	BackendTls *BackendTls `toml:"backend_tls" json:"backend_tls"`

	// Optional configuration for protocol = udp
	Udp *Udp `toml:"udp" json:"udp"`

	// Access configuration
	Access *AccessConfig `toml:"access" json:"access"`

	// Discovery configuration
	Discovery *DiscoveryConfig `toml:"discovery" json:"discovery"`

	// Healthcheck configuration
	Healthcheck *HealthcheckConfig `toml:"healthcheck" json:"healthcheck"`
}

/**
 * Common part of Tls and BackendTls types
 */
type tlsCommon struct {
	Ciphers             []string `toml:"ciphers" json:"ciphers"`
	PreferServerCiphers bool     `toml:"prefer_server_ciphers" json:"prefer_server_ciphers"`
	MinVersion          string   `toml:"min_version" json:"min_version"`
	MaxVersion          string   `toml:"max_version" json:"max_version"`
	SessionTickets      bool     `toml:"session_tickets" json:"session_tickets"`
}

/**
 * Server Tls options
 * for protocol = "tls"
 */
type Tls struct {
	CertPath string `toml:"cert_path" json:"cert_path"`
	KeyPath  string `toml:"key_path" json:"key_path"`
	tlsCommon
}

type BackendTls struct {
	RootCaCertPath *string `toml:"root_ca_cert_path" json:"root_ca_cert_path"`
	CertPath       *string `toml:"cert_path" json:"cert_path"`
	KeyPath        *string `toml:"key_path" json:"key_path"`
	tlsCommon
}

/**
 * Server udp options
 * for protocol = "udp"
 */
type Udp struct {
	MaxResponses int `toml:"max_responses" json:"max_responses"`
}

/**
 * Access configuration
 */
type AccessConfig struct {
	Default string   `toml:"default" json:"default"`
	Rules   []string `toml:"rules" json:"rules"`
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
	*ConsulDiscoveryConfig
}

type StaticDiscoveryConfig struct {
	StaticList []string `toml:"static_list" json:"static_list"`
}

type SrvDiscoveryConfig struct {
	SrvLookupServer  string `toml:"srv_lookup_server" json:"srv_lookup_server"`
	SrvLookupPattern string `toml:"srv_lookup_pattern" json:"srv_lookup_pattern"`
	SrvDnsProtocol   string `toml:"srv_dns_protocol" json:"srv_dns_protocol"`
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
	DockerContainerHostEnvVar  string `toml:"docker_container_host_env_var" json:"docker_container_host_env_var"`

	DockerTlsEnabled    bool   `toml:"docker_tls_enabled" json:"docker_tls_enabled"`
	DockerTlsCertPath   string `toml:"docker_tls_cert_path" json:"docker_tls_cert_path"`
	DockerTlsKeyPath    string `toml:"docker_tls_key_path" json:"docker_tls_key_path"`
	DockerTlsCacertPath string `toml:"docker_tls_cacert_path" json:"docker_tls_cacert_path"`
}

type ConsulDiscoveryConfig struct {
	ConsulHost               string `toml:"consul_host" json:"consul_host"`
	ConsulServiceName        string `toml:"consul_service_name" json:"consul_service_name"`
	ConsulServiceTag         string `toml:"consul_service_tag" json:"consul_service_tag"`
	ConsulServicePassingOnly bool   `toml:"consul_service_passing_only" json:"consul_service_passing_only"`
	ConsulDatacenter         string `toml:"consul_datacenter" json:"consul_datacenter"`

	ConsulAuthUsername string `toml:"consul_auth_username" json:"consul_auth_username"`
	ConsulAuthPassword string `toml:"consul_auth_password" json:"consul_auth_password"`

	ConsulTlsEnabled    bool   `toml:"consul_tls_enabled" json:"consul_tls_enabled"`
	ConsulTlsCertPath   string `toml:"consul_tls_cert_path" json:"consul_tls_cert_path"`
	ConsulTlsKeyPath    string `toml:"consul_tls_key_path" json:"consul_tls_key_path"`
	ConsulTlsCacertPath string `toml:"consul_tls_cacert_path" json:"consul_tls_cacert_path"`
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
