<img src="/logo.png?raw=true" alt="gobetween" width="256px" />

[![Build Status](https://travis-ci.org/yyyar/gobetween.svg?branch=master)](https://travis-ci.org/yyyar/gobetween)

**gobetween** -  modern & minimalistic proxy server for the :cloud: Cloud era.

**Current status**: *In development*.

Incompatible changes may occur until v1.0.0. gobetween currently is not production ready, but we already successfully using it in several highy loaded production environments.

## Features

* TCP Proxy (udp and more will come later)

* Clear and beautiful TOML config file.

* Discovery
  * **Static** - hardcode backends in config file
  * **Docker** - query backends from Docker / Swarm API filtered by label
  * **Exec** - execte arbitrary program and read backends from it's output
  * **JSON** - make http query and parse backends from response json
  * **Plaintext** - make http query and parse backends from response text with regexps
  * **SRV** - query SRV server for a backends

* Healthchecks
  * **Ping** - simple TCP ping healtcheck
  * **Exec** - execute external program passing host & port, and read healtcheck status from the stdout

* Balancing
  * **Iphash**
  * **Leastconn**
  * **Roundrobin**
  * **Weight**

* Integrates seamlessly with Docker (thanks to docker discovery) and with any custom system (thanks to exec discovery and healtchecks)!

## Usage

* [Download and Install](https://github.com/yyyar/gobetween/wiki/2.-Installation)
* Consider [Configuration manual](https://github.com/yyyar/gobetween/wiki/3.-Configuration) and [config file](config/gobetween.toml)


## Hacking

* Install Go 1.6+ https://golang.org/
* `$ git clone git@github.com:yyyar/gobetween.git`
* `$ make deps`
* `$ make run`

### Debug and test
Run several web servers for test in different terminals:

* `$ python -m SimpleHTTPServer 8000`
* `$ python -m SimpleHTTPServer 8001`

Put `localhost:8000` and `localhost:8001` to static_list of static discovery in config file, then test:

* `$ curl http://localhost:3000`

## Performance
See [Performance Testing](https://github.com/yyyar/gobetween/wiki/Performance-tests-results).


## Authors & Contributors
- [Yaroslav Pogrebnyak](http://pogrebnyak.info)
- Nick Doikov
- Ievgen Ponomarenko


## The Name
It's play on words: gobetween ("go between"). Also it's written in Go,
and it's a proxy so it's between 2 parties :-)


## License
MIT. See LICENSE file for more details.


## Logo
Logo by [Max Demchenko](https://www.linkedin.com/in/max-demchenko-116170112)
