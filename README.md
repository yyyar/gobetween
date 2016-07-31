<img src="/logo.png?raw=true" alt="gobetween" width="256px" />

[![Tag](https://img.shields.io/github/tag/yyyar/gobetween.svg)](https://github.com/yyyar/gobetween/releases/latest)
[![Build Status](https://travis-ci.org/yyyar/gobetween.svg?branch=master)](https://travis-ci.org/yyyar/gobetween)
[![Go Report Card](https://goreportcard.com/badge/github.com/yyyar/gobetween)](https://goreportcard.com/report/github.com/yyyar/gobetween)
[![Docs](https://img.shields.io/badge/docs-current-brightgreen.svg)](https://github.com/yyyar/gobetween/wiki)
[![Docker](https://img.shields.io/docker/pulls/yyyar/gobetween.svg)](https://hub.docker.com/r/yyyar/gobetween/)
[![License](https://img.shields.io/badge/license-MIT-blue.svg)](/LICENSE)
[![Gitter](https://badges.gitter.im/yyyar/gobetween.svg)](https://gitter.im/yyyar/gobetween?utm_source=badge&utm_medium=badge&utm_campaign=pr-badge)

**gobetween** -  modern & minimalistic proxy server for the :cloud: Cloud era.

**Current status**: *Under active development*. Currently in use in several highy loaded production environments.

## Features

* [L4 TCP Load Balancing](https://github.com/yyyar/gobetween/wiki)

* [Clear Configuration](https://github.com/yyyar/gobetween/wiki/Configuration) with [TOML](config/gobetween.toml) or [JSON](config/gobetween.json)
  * **File** - read configuration from the file
  * **URL** - querying URL by HTTP and get configuration from response 
  * **Consul** - get configuration from Consul key-value storage value

* [Management REST API] (https://github.com/yyyar/gobetween/wiki/REST-API)
  * System Information
  * Configuration Dump
  * Servers List / Create / Delete
  * Servers and Backends rx/tx, Status, Active Connections, etc.
 
* [Discovery](https://github.com/yyyar/gobetween/wiki/Discovery)
  * **Static** - hardcode backends list in config file
  * **Docker** - query backends from Docker / Swarm API filtered by label
  * **Exec** - execte arbitrary program and get backends from it's stdout
  * **JSON** - query arbitrary http url and pick backends from response json (of any structure)
  * **Plaintext** - query arbitrary http and parse backends from response text with customized regexp
  * **SRV** - query DNS server and get backends from SRV records

* [Healthchecks](https://github.com/yyyar/gobetween/wiki/Healthchecks)
  * **Ping** - simple TCP ping healtcheck
  * **Exec** - execute arbitrary program passing host & port as options, and read healtcheck status from the stdout

* [Balancing Strategies](https://github.com/yyyar/gobetween/wiki/Balancing)
  * **Iphash**
  * **Leastconn**
  * **Roundrobin**
  * **Weight**

* Integrates seamlessly with Docker and with any custom system (thanks to Exec discovery and healtchecks)

* Single binary distribution


## Architecture
<img src="http://i.piccy.info/i9/8b92154435be32f21eaa3ff7b3dc6d1c/1466244332/74457/1043487/gog.png" alt="gobetween" />

## Usage

* [Download and Install](https://github.com/yyyar/gobetween/wiki/Installation)
* Consider [Configuration manual](https://github.com/yyyar/gobetween/wiki) and [config file](config/gobetween.toml)


## Hacking

* Install Go 1.6+ https://golang.org/
* `$ git clone git@github.com:yyyar/gobetween.git`
* `$ make deps`
* `$ make run`

### Debug and Test
Run several web servers for tests in different terminals:

* `$ python -m SimpleHTTPServer 8000`
* `$ python -m SimpleHTTPServer 8001`

Put `localhost:8000` and `localhost:8001` to static_list of static discovery in config file, then try it:

* `$ gobetween -c gobetween.toml`

* `$ curl http://localhost:3000`

## Performance
See [Performance Testing](https://github.com/yyyar/gobetween/wiki/Performance-tests)

## The Name
It's play on words: gobetween ("go between"). 

Also, it's written in Go, and it's a proxy so it's something that stays between 2 parties :smile:

## License
MIT. See LICENSE file for more details.

## Authors & Contributors
- [Yaroslav Pogrebnyak](http://pogrebnyak.info)
- Nick Doikov
- Ievgen Ponomarenko

## Logo
Logo by [Max Demchenko](https://www.linkedin.com/in/max-demchenko-116170112)
