package proxyprotocol

import (
	"net"

	proxyproto "github.com/pires/go-proxyproto"
)

/// SendProxyProtocolV1 sends a proxy protocol v1 header to initialize the connection
/// https://www.haproxy.org/download/1.8/doc/proxy-protocol.txt
func SendProxyProtocolV1(client net.Conn, backend net.Conn) error {
	sourceIP := client.RemoteAddr()

	destinationIP := client.LocalAddr()

	h := proxyproto.Header{
		Version:           1,
		SourceAddr:        sourceIP,
		TransportProtocol: '\x11',
		DestinationAddr:   destinationIP,
	}

	_, err := h.WriteTo(backend)
	if err != nil {
		return nil
	}

	return nil
}
