package test

import (
	"bytes"
	"encoding/binary"
	"io/ioutil"
	"net"
	"strconv"
	"testing"

	"github.com/yyyar/gobetween/utils/proxyprotocol"
)

func testSendProxyProtocol(t *testing.T, addr string, version string) (serverPort, clientPort string, received []byte) {
	listener, err := net.Listen("tcp", addr+":0")
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()

	_, serverPort, err = net.SplitHostPort(listener.Addr().String())
	if err != nil {
		t.Fatal(err)
	}

	go func() {
		client, err := net.Dial("tcp", addr+":"+serverPort)
		if err != nil {
			t.Fatal(err)
		}
		defer client.Close()

		_, clientPort, err = net.SplitHostPort(client.LocalAddr().String())
		if err != nil {
			t.Fatal(err)
		}

		switch version {
		case "1":
			proxyprotocol.SendProxyProtocolV1(client, client)
		default:
			t.Fatalf("Unsupported proxy_protocol version " + version + ", aborting connection")
		}
	}()

	server, err := listener.Accept()
	if err != nil {
		t.Fatal(err)
	}
	defer server.Close()

	buf, err := ioutil.ReadAll(server)
	if err != nil {
		t.Fatal(err)
	}

	received = []byte(buf)

	return serverPort, clientPort, received
}

func TestSendProxyProtocolV1IPv4(t *testing.T) {
	serverPort, clientPort, received := testSendProxyProtocol(t, "127.0.0.1", "1")

	expected := "PROXY TCP4 127.0.0.1 127.0.0.1 " + serverPort + " " + clientPort + "\r\n"
	if string(received) != expected {
		t.Fatalf("%s != %s", string(received), expected)
	}
}

func TestSendProxyProtocolV1IPv6(t *testing.T) {
	serverPort, clientPort, received := testSendProxyProtocol(t, "[::1]", "1")

	expected := "PROXY TCP6 ::1 ::1 " + serverPort + " " + clientPort + "\r\n"
	if string(received) != expected {
		t.Fatalf("%s != %s", string(received), expected)
	}
}
