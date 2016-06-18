<img src="/logo.png?raw=true" alt="gobetween" width="256px" />

[![Tag](https://img.shields.io/github/tag/yyyar/gobetween.svg)](https://github.com/yyyar/gobetween/releases/latest)
[![Build Status](https://travis-ci.org/yyyar/gobetween.svg?branch=master)](https://travis-ci.org/yyyar/gobetween)
[![Go Report Card](https://goreportcard.com/badge/github.com/yyyar/gobetween)](https://goreportcard.com/report/github.com/yyyar/gobetween)
[![Docs](https://img.shields.io/badge/docs-current-brightgreen.svg)](https://github.com/yyyar/gobetween/wiki)
[![Docker](https://img.shields.io/docker/pulls/yyyar/gobetween.svg)](https://hub.docker.com/r/yyyar/gobetween/)
[![License](https://img.shields.io/badge/license-MIT-blue.svg)](/LICENSE)

**gobetween** -  modern & minimalistic proxy server for the :cloud: Cloud era.

**Current status**: *Under active development*. Currently in use in several highy loaded production environments.

## Features

* TCP Load Balancing

* Clear and beautiful TOML config file.

* Backends Discovery
  * **Static** - hardcode backends list in config file
  * **Docker** - query backends from Docker / Swarm API filtered by label
  * **Exec** - execte arbitrary program and get backends from it's stdout
  * **JSON** - query arbitrary http url and pick backends from response json (of any structure)
  * **Plaintext** - query arbitrary http and parse backends from response text with customized regexp
  * **SRV** - query DNS server and get backends from SRV records

* Backends Healthchecks
  * **Ping** - simple TCP ping healtcheck
  * **Exec** - execute arbitrary program passing host & port as options, and read healtcheck status from the stdout

* Balancing Strategies
  * **Iphash**
  * **Leastconn**
  * **Roundrobin**
  * **Weight**

* Integrates seamlessly with Docker and with any custom system (thanks to exec discovery and healtchecks)!

## Usage

* [Download and Install](https://github.com/yyyar/gobetween/wiki/2.-Installation)
* Consider [Configuration manual](https://github.com/yyyar/gobetween/wiki/3.-Configuration) and [config file](config/gobetween.toml)


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

* `$ curl http://localhost:3000`

## Performance
See [Performance Testing](https://github.com/yyyar/gobetween/wiki/Performance-tests-results).

## The Name
It's play on words: gobetween ("go between"). Also, it's written in Go, and it's a proxy so it's something that stays between 2 parties :smile:

## License
MIT. See LICENSE file for more details.

## Authors & Contributors
- [Yaroslav Pogrebnyak](http://pogrebnyak.info)
- Nick Doikov
- Ievgen Ponomarenko

## Logo
Logo by [Max Demchenko](https://www.linkedin.com/in/max-demchenko-116170112)
