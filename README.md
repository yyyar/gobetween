[![Stand With Ukraine](https://raw.githubusercontent.com/vshymanskyy/StandWithUkraine/main/banner-direct.svg)](https://vshymanskyy.github.io/StandWithUkraine)

<img src="/logo.png?raw=true" alt="gobetween" width="256px" />

[![Tag](https://img.shields.io/github/tag/yyyar/gobetween.svg)](https://github.com/yyyar/gobetween/releases/latest)
[![Build Status](https://travis-ci.org/yyyar/gobetween.svg?branch=master)](https://travis-ci.org/yyyar/gobetween)
[![Go Report Card](https://goreportcard.com/badge/github.com/yyyar/gobetween)](https://goreportcard.com/report/github.com/yyyar/gobetween)
[![Docs](https://img.shields.io/badge/docs-current-brightgreen.svg)](https://github.com/yyyar/gobetween/wiki)
[![Docker](https://img.shields.io/docker/pulls/yyyar/gobetween.svg)](https://hub.docker.com/r/yyyar/gobetween/)
[![Snap Status](https://build.snapcraft.io/badge/yyyar/gobetween.svg)](https://build.snapcraft.io/user/yyyar/gobetween)
[![Telegram](https://img.shields.io/badge/telegram-chat-blue.svg)](https://t.me/joinchat/GdlUlg_gRfchk1BORU82PA)
[![License](https://img.shields.io/badge/license-MIT-blue.svg)](/LICENSE)


**gobetween** -  modern & minimalistic load balancer and reverse-proxy for the :cloud: Cloud era.

**Current status**: *Maintenance mode, accepting PRs*. Currently in use in several highly loaded production environments.

## Features

* [Fast L4 Load Balancing](https://github.com/yyyar/gobetween/wiki)
  * **TCP** - with optional [The PROXY Protocol](https://github.com/yyyar/gobetween/wiki/Proxy-Protocol) support
  * **TLS** - [TLS Termination](https://github.com/yyyar/gobetween/wiki/Protocols#tls) + [ACME](https://github.com/yyyar/gobetween/wiki/Protocols#tls) & [TLS Proxy](https://github.com/yyyar/gobetween/wiki/Tls-Proxying)
  * **UDP** - with optional virtual sessions and transparent mode


* [Clear & Flexible Configuration](https://github.com/yyyar/gobetween/wiki/Configuration) with [TOML](config/gobetween.toml) or [JSON](config/gobetween.json)
  * **File** - read configuration from the file
  * **URL** - query URL by HTTP and get configuration from the response body 
  * **Consul** - query Consul key-value storage API for configuration

* [Management REST API](https://github.com/yyyar/gobetween/wiki/REST-API)
  * **System Information** - general server info
  * **Configuration** - dump current config 
  * **Servers** - list, create & delete
  * **Stats & Metrics** - for servers and backends including rx/tx, status, active connections & etc.
 
* [Discovery](https://github.com/yyyar/gobetween/wiki/Discovery)
  * **Static** - hardcode backends list in the config file
  * **Docker** - query backends from Docker / Swarm API filtered by label
  * **Exec** - execute an arbitrary program and get backends from its stdout
  * **JSON** - query arbitrary http url and pick backends from response json (of any structure)
  * **Plaintext** - query arbitrary http and parse backends from response text with customized regexp
  * **SRV** - query DNS server and get backends from SRV records
  * **Consul** - query Consul Services API for backends 
  * **LXD** - query backends from LXD

* [Healthchecks](https://github.com/yyyar/gobetween/wiki/Healthchecks)
  * **Ping** - simple TCP ping healthcheck
  * **Exec** - execute arbitrary program passing host & port as options, and read healthcheck status from the stdout
  * **Probe** - send specific bytes to backend (udp, tcp or tls) and expect a correct answer (bytes or regexp)

* [Balancing Strategies](https://github.com/yyyar/gobetween/wiki/Balancing) (with [SNI](https://github.com/yyyar/gobetween/wiki/Server-Name-Indication) support)
  * **Weight** - select backend from pool based relative weights of backends
  * **Roundrobin** - simple elect backend from pool in circular order
  * **Iphash** - route client to the same backend based on client ip hash
  * **Iphash1** - same as iphash but backend removal consistent (clients remain connecting to the same backend, even if some other backends down)
  * **Leastconn** - select backend with least active connections
  * **Leastbandwidth** -  backends with least bandwidth

* Integrates seamlessly with Docker and with any custom system (thanks to Exec discovery and healthchecks)

* Single binary distribution


## Architecture
<img src="http://i.piccy.info/i9/8b92154435be32f21eaa3ff7b3dc6d1c/1466244332/74457/1043487/gog.png" alt="gobetween" />

## Usage

* Install with snap: https://snapcraft.io/gobetween
* [Other Installation Options](https://github.com/yyyar/gobetween/wiki/Installation)
* [Read Configuration Reference](https://github.com/yyyar/gobetween/wiki)
* Execute `gobetween --help` for full help on all available commands and options.

## Hacking

* Install Go 1.14+ https://golang.org/
* `$ git clone git@github.com:yyyar/gobetween.git`
* `$ make`
* `$ make run`

### Debug and Test
Run several web servers for tests in different terminals:

* `$ python -m SimpleHTTPServer 8000`
* `$ python -m SimpleHTTPServer 8001`

Instead of Python's internal HTTP module, you can also use a single binary (Go based) webserver like:
https://github.com/udhos/gowebhello

**gowebhello** has support for SSL sertificates as well (**HTTPS** mode), in case you want to do quick demos
of the **TLS+SNI** capabilities of gobetween.

Put `localhost:8000` and `localhost:8001` to `static_list` of static discovery in config file, then try it:

* `$ gobetween -c gobetween.toml`

* `$ curl http://localhost:3000`

Enable [profiler](https://blog.golang.org/profiling-go-programs) and debug issues you encounter
```
[profiler]
enabled = true     # false | true
bind    = ":6060"  # "host:port"
```

## Performance
It's Fast! See [Performance Testing](https://github.com/yyyar/gobetween/wiki/Performance-tests)

## The Name
It's a play on words: gobetween ("go between"). 

Also, it's written in Go, and it's a proxy so it's something that stays between 2 parties :smile:

## License
MIT. See LICENSE file for more details.

## Authors & Maintainers
- [Yaroslav Pogrebnyak](http://pogrebnyak.info)
- [Nick Doikov](https://github.com/nickdoikov)
- [Ievgen Ponomarenko](https://github.com/kikom)
- [Illarion Kovalchuk](https://github.com/illarion)

## All Contributors
- See [AUTHORS](AUTHORS)

## Community
- Join gobetween Telegram group [here](https://t.me/joinchat/GdlUlg_gRfchk1BORU82PA).

## Logo
Logo by [Max Demchenko](https://www.linkedin.com/in/max-demchenko-116170112)
