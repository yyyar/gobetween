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
		case "2":
			proxyprotocol.SendProxyProtocolV2(client, client)
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

func TestSendProxyProtocolV2IPv4(t *testing.T) {
	serverPort, clientPort, received := testSendProxyProtocol(t, "127.0.0.1", "2")

	serverPortInt, _ := strconv.Atoi(serverPort)
	clientPortInt, _ := strconv.Atoi(clientPort)

	expected := new(bytes.Buffer)
	expected.Write([]byte{13, 10, 13, 10, 0, 13, 10, 81, 85, 73, 84, 10, 32, 17, 0, 12, 127, 0, 0, 1, 127, 0, 0, 1})
	binary.Write(expected, binary.BigEndian, uint16(serverPortInt))
	binary.Write(expected, binary.BigEndian, uint16(clientPortInt))

	if bytes.Compare(received, expected.Bytes()) != 0 {
		t.Fatalf("%v != %v", received, expected.Bytes())
	}
}

func TestSendProxyProtocolV2IPv6(t *testing.T) {
	serverPort, clientPort, received := testSendProxyProtocol(t, "[::1]", "2")

	serverPortInt, _ := strconv.Atoi(serverPort)
	clientPortInt, _ := strconv.Atoi(clientPort)

	expected := new(bytes.Buffer)
	expected.Write([]byte{13, 10, 13, 10, 0, 13, 10, 81, 85, 73, 84, 10, 32, 33, 0, 36, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1})
	binary.Write(expected, binary.BigEndian, uint16(serverPortInt))
	binary.Write(expected, binary.BigEndian, uint16(clientPortInt))

	if bytes.Compare(received, expected.Bytes()) != 0 {
		t.Fatalf("%v != %v", received, expected.Bytes())
	}
}
