# Changelog

## [0.2.0] - 2016-07-22
This release brings several big features such as full-functional REST API and Stats, as well
as may bugfixes and improvements. All changes are backward-compatible with 0.1.0.

### New Features
- REST API implementation (info, servers list/create/remove, stats, config dump).
- Implemented gathering stats for servers and backends (rx/tx, rx/tx per second, connections count, etc)

### Added
- Set GOMAXPROCS to cpu count automatically if no env var is present
- Added TLS support for Docker discovery
- Added docker_container_host_env_var property to Docker discovery
- Allow any type of value (int or string) in port in JSON discovery
- Make healthchecks optional

### Fixes
- Fixed panic runtime error exec discovery when exec_command is not valid path and timeout=0
- Fixed roundrobin balance strategy
- Fixed how SRV discovery handler large UDP responses; Fixed sometimes missed port.
- Fixed parsing backend on windows (with \r newlines)


## [0.1.0] - 2016-06-08
### Added
- Initial project implementation (by @yyyar and @kikom).
