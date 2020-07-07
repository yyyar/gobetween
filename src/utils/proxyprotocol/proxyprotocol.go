package proxyprotocol

import (
	"fmt"
	"net"
	"strconv"

	proxyproto "github.com/pires/go-proxyproto"
	"github.com/yyyar/gobetween/config"
	"github.com/yyyar/gobetween/core"
)

func addrToIPAndPort(addr net.Addr) (ip net.IP, port uint16, err error) {
	ipString, portString, err := net.SplitHostPort(addr.String())
	if err != nil {
		return
	}

	ip = net.ParseIP(ipString)
	if ip == nil {
		err = fmt.Errorf("Could not parse IP")
		return
	}

	p, err := strconv.ParseInt(portString, 10, 64)
	if err != nil {
		return
	}
	port = uint16(p)
	return
}

/// SendProxyProtocolV1 sends a proxy protocol v1 header to initialize the connection
/// https://www.haproxy.org/download/1.8/doc/proxy-protocol.txt
func SendProxyProtocolV1(cfg *config.ProxyProtocol, ctx *core.TcpContext, backend net.Conn) error {
	client := ctx.Conn
	sourceIP, sourcePort, err := addrToIPAndPort(client.RemoteAddr())
	if err != nil {
		return err
	}

	destinationIP, destinationPort, err := addrToIPAndPort(client.LocalAddr())
	if err != nil {
		return err
	}

	h := proxyproto.Header{
		Version:            1,
		SourceAddress:      sourceIP,
		SourcePort:         sourcePort,
		DestinationAddress: destinationIP,
		DestinationPort:    destinationPort,
	}
	if sourceIP.To4() != nil {
		h.TransportProtocol = proxyproto.TCPv4
	} else {
		h.TransportProtocol = proxyproto.TCPv6
	}

	_, err = h.WriteTo(backend)
	if err != nil {
		return nil
	}
	return nil
}

/// SendProxyProtocolV2 sends a proxy protocol v2 header to initialize the connection
/// https://www.haproxy.org/download/1.8/doc/proxy-protocol.txt
func SendProxyProtocolV2(cfg *config.ProxyProtocol, ctx *core.TcpContext, backend net.Conn) error {
	client := ctx.Conn
	sourceIP, sourcePort, err := addrToIPAndPort(client.RemoteAddr())
	if err != nil {
		return err
	}

	destinationIP, destinationPort, err := addrToIPAndPort(client.LocalAddr())
	if err != nil {
		return err
	}

	h := proxyproto.Header{
		Version:            2,
		Command:            proxyproto.PROXY,
		SourceAddress:      sourceIP,
		SourcePort:         sourcePort,
		DestinationAddress: destinationIP,
		DestinationPort:    destinationPort,
	}
	if sourceIP.To4() != nil {
		h.TransportProtocol = proxyproto.TCPv4
	} else {
		h.TransportProtocol = proxyproto.TCPv6
	}

	if ctx.TlsState != nil {
		// SSL TLV should be appended to header here, but go-proxyproto current
		// version does not include any method that performs this operation
	}

	_, err = h.WriteTo(backend)
	if err != nil {
		return nil
	}
	return nil
}
