 <img src="/logo.png?raw=true" alt="gobetween" width="256px" />

[![Build Status](https://travis-ci.org/yyyar/gobetween.svg?branch=master)](https://travis-ci.org/yyyar/gobetween)

**gobetween** -  modern & minimalistic proxy server for the :cloud: Cloud era.

**Current status**: *In development*. Incompatible changes may occur until v1.0.0. gobetween is still not production ready, but we already successfully tried it in several highy loaded production deployments.

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

* Integrates seamlessly with Docker (thanks yo docker discovery) and with any custom syste (thanks to exec discovery and healchecks)!

## Usage

* Download and install https://github.com/yyyar/gobetween/releases

For configuration documentation see [wiki](https://github.com/yyyar/gobetween/wiki) and [config file](config/gobetween.toml).


## Hacking

### Requirements
* Go 1.6+ https://golang.org/

### Clone
* `$ git clone git@github.com:yyyar/gobetween.git`

### Install dependencies
* `$ make deps`

### Run
* `$ make run`

### Debug
Run several web servers for test in different terminals:
* `$ python -m SimpleHTTPServer 8000`
* `$ python -m SimpleHTTPServer 8001`

### Test:
* `$ curl http://localhost:3000`


## Install from sources
* `$ git clone git@github.com:yyyar/gobetween.git`
* `$ make`
* `$ sudo -E make install`
* `$ vim /etc/gobetween.toml`
* `$ gobetween -c /etc/gobetween.toml`

### Uninstall
* `$ sudo make uninstall`


## Configuration
For details see [wiki](https://github.com/yyyar/gobetween/wiki) and [config/gobetween.toml](config/gobetween.toml)


## Performance
To increase performance run with:
```GOMAXPROCS=`nproc` gobetween```

See [Performance Testing](https://github.com/yyyar/gobetween/wiki/Performance-tests-results) for performance testing results.


## Authors & Contributors
- [Yaroslav Pogrebnyak](http://pogrebnyak.info)
- Nick Doikov
- Ievgen Ponomarenko


## The Name
It's play on words: gobetween ("go between"). ALso it's written in Go,
and it's a proxy so it's between 2 parties :-)


## License
MIT. See LICENSE file for more details.


## Logo
Logo by [Max Demchenko](https://www.linkedin.com/in/max-demchenko-116170112)
