 <img src="/logo.png?raw=true" alt="gobetween" width="256px" />
 
**gobetween** - modern & minimalistic proxy server for the Cloud era.

*Current status*
In development. Incompatible changes may occur until v1.0.0.
If you tried gobetween in your dev, your feedback would be highly appreciated.

## Features

TCP proxying (udp and more will come later).

Flexible backends discovery strategies:
* Static list - hardcoded list
* Docker / Docker Swarm - query backends from Docker API by label
* Exec - Run arbitrary external script and read backends from output
* JSON - Make http query and parse json
* Text/Regexp - Make http query and parse text with regexps
* SRV - Query SRV server

Powerful healthchecks:
* Ping - Simple TCP ping healtcheck
* Exec - Run external script providing host and port and read healtcheck status from output

Balancing strategies:
* Iphash
* Leastconn
* Roundrobin
* Weight

Clear and beautiful TOML config file.

## Usage (for end users)

# Download and install
* https://github.com/yyyar/gobetween/releases
* 
For configuration documentation see default config file.

## Development (for hacking)

### Requirements
* Go 1.6+ https://golang.org/

### Install dependencies
* $ make deps

### Debug and Test
Run several web servers for test in different terminals:
* `$ python -m SimpleHTTPServer 8000`
* `$ python -m SimpleHTTPServer 8001`

Run
* `$ make run`

Test with curl:
* `$ curl http://localhost:3000`

# Install from sources
* Clone this repo
* `$ make`
* `$ sudo -E make install`
* `$ vim /etc/gobetween.toml`
* `$ gobetween -c /etc/gobetween.toml`

# Uninstall
* `$ sudo make uninstall`

## Configuration
For details see [config/gobetween.toml](config/gobetween.toml)

## Performance
To increase performance run with:
```GOMAXPROCS=`nproc` gobetween```

## Authors & Constributors
- [Yaroslav Pogrebnyak](http://pogrebnyak.info)
- Nick Doikov
- Ievgen Ponomarenko

## The Name
It's play on words: gobetween ("go between"). ALso it's written in Go,
and it's a proxy so it's between 2 parties :-)

## License
MIT. See LICENSE file for more details.

## Logo
Logo was designed by [Max Demchenko](https://www.linkedin.com/in/max-demchenko-116170112)
