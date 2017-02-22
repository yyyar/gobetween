#
# Makefile
# @author Yaroslav Pogrebnyak <yyyaroslav@gmail.com>
# @author Ievgen Ponomarenko <kikomdev@gmail.com>
#

.PHONY: update clean build build-all run package deploy test authors dist

export GOPATH := ${PWD}/vendor:${PWD}
export GOBIN := ${PWD}/vendor/bin


NAME := gobetween
VERSION := $(shell cat VERSION)
LDFLAGS := "-X main.version=${VERSION}"

default: build

clean:
	@echo Cleaning up...
	@rm bin/* -f
	@rm dist/* -f
	@echo Done.

build:
	@echo Building...
	go build -v -o ./bin/$(NAME) -ldflags ${LDFLAGS} ./src/*.go
	@echo Done.

build-static:
	@echo Building...
	CGO_ENABLED=0 go build -v -o ./bin/$(NAME) -ldflags '-s -w --extldflags "-static”  ${LDFLAGS}' ./src/*.go
	@echo Done.

run: build
	./bin/$(NAME) -c ./config/${NAME}.toml

test:
	@go test test/*.go

install: build
	install -d ${DESTDIR}/usr/local/bin/
	install -m 755 ./bin/${NAME} ${DESTDIR}/usr/local/bin/${NAME}
	install ./config/${NAME}.toml ${DESTDIR}/etc/${NAME}.toml

uninstall:
	rm -f ${DESTDIR}/usr/local/bin/${NAME}
	rm -f ${DESTDIR}/etc/${NAME}.toml

authors:
	@git log --format='%aN <%aE>' | LC_ALL=C.UTF-8 sort | uniq -c | sort -nr | sed "s/^ *[0-9]* //g" > AUTHORS
	@cat AUTHORS

clean-deps:
	rm -rf ./vendor/src
	rm -rf ./vendor/pkg
	rm -rf ./vendor/bin

deps: clean-deps
	GOPATH=${PWD}/vendor go get -u -v \
	github.com/BurntSushi/toml \
	github.com/miekg/dns \
	github.com/fsouza/go-dockerclient \
	github.com/Sirupsen/logrus \
	github.com/elgs/gojq \
	gopkg.in/gin-gonic/gin.v1 \
	github.com/hashicorp/consul/api \
	github.com/spf13/cobra \
	github.com/laher/goxc \
	gopkg.in/gin-contrib/cors.v1

clean-dist:
	rm -rf ./dist/${VERSION}

dist:
	@echo Building for all platforms ...
	./vendor/bin/goxc -d="./dist" \
		-tasks=xc,archive \
		-arch="386 amd64" \
		-pv="${VERSION}" \
		-os="linux windows darwin" \
		-include="README.md,LICENSE,CHANGELOG,VERSION,config/gobetween.toml,share" \
		-build-ldflags=${LDFLAGS}
	rm ./debian -rf
	@echo Done.

build-container-latest: build 
	@echo Building docker container LATEST
	docker build -t yyyar/gobetween .

build-container-tagged: build 
	@echo Building docker container ${VERSION}
	docker build -t yyyar/gobetween:${VERSION} .
