# Changelog

## [0.4.0] - 2017-04-07
This release brings many new features and improvemets, as well as bugfixes.
Major things are UDP support, TLS termination, TLS proxy, SNI-aware balancing.

### New Features
- UDP protocol support
- TLS termination
- TLS proxy (connect to backends with TLS and configurable certs)
- SNI-aware balancing (routing based on hostname from TLS Server Name Indication record)

### Added
- Possibility to enable CORS for REST API

### Fixed
- Messed up `client_idle_timeout` and `backend_idle_timeout`
- Bugs in balancers: iphash, roundrobin, weight - now work more accurately
- Goroutine/memory leak caused by consul discovery not reusing http client

### Changed
- Docker discovery now can have empty TLS certificates.
- Migrated to golang 1.8. Now it's minimal requirement for the build.



## [0.3.0] - 2016-08-18
This release brings several new features and improvemets, as well as bugfixes. Major things are
integrations with Consul, more flexible command-line options and Access control module.

### New Features
- Consul Discovery
- Ability to load config not only from file, but also from URL and Consul key-value storage on startup
- More powerful command-line interface
- Leastbandwidth balancing strategy

### Added
- Allow passing parameters as GOBETWEEN env variable instead of args
- Possibility to specify format in /dump endpoint (toml or json)
- Refused connections counters for backends
- TCP mode for DNS SRV Discovery

### Fixed
- Creating server with the same name via rest api causes api to freeze
- Runtime error when no [default] section is present in config

### Changed
- Replaced big.Int with uint64 for simplicity and performance reasons.



## [0.2.0] - 2016-07-22
This release brings several big features such as full-functional REST API and Stats, as well
as may bugfixes and improvements. All changes are backward-compatible with 0.1.0.

### New Features
- REST API implementation (info, servers list/create/remove, stats, config dump).
- Implemented gathering stats for servers and backends (rx/tx, rx/tx per second, connections count, etc)

### Added
- Set GOMAXPROCS to cpu count automatically if no env var is present
- Added TLS support for Docker discovery
- Added `docker_container_host_env_var` property to Docker discovery
- Allow any type of value (int or string) in port in JSON discovery
- Make healthchecks optional

### Fixed
- Fixed panic runtime error exec discovery when `exec_command` is not valid path and timeout=0
- Fixed roundrobin balance strategy
- Fixed how SRV discovery handler large UDP responses; Fixed sometimes missed port.
- Fixed parsing backend on windows (with \r newlines)


## [0.1.0] - 2016-06-08
### Added
- Initial project implementation (by @yyyar and @kikom).
