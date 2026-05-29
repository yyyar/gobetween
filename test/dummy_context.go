package test

import (
	"fmt"
	"net"
)

type DummyContext struct {
	ip   net.IP
	port int
}

func (d DummyContext) String() string {
	return fmt.Sprintf("%v:%v", d.Ip(), d.Port())
}

func (d DummyContext) Ip() net.IP {
	if d.ip == nil {
		d.ip = make(net.IP, 1)
	}
	return d.ip
}

func (d DummyContext) Port() int {
	return d.port
}

func (d DummyContext) Sni() string {
	return ""
}
