package test

import (
	"../src/balance"
	"../src/core"
	"errors"
	"fmt"
	"math/rand"
	"testing"
)

func makeDistribution(balancer core.Balancer, backends []*core.Backend, clients []DummyContext) (map[string]*core.Backend, error) {

	result := make(map[string]*core.Backend)

	for _, client := range clients {
		electedBackend, err := balancer.Elect(client, backends)

		if err != nil {
			return nil, err
		}

		if electedBackend == nil {
			return nil, errors.New("Elected nil backend!")
		}

		result[client.ip.String()] = electedBackend
	}

	return result, nil

}

// Prepare list of backends, for testing purposes they end with .1, .2, .3 etc
// It will be easier to print them if needed
func prepareBackends(base string, n int) []*core.Backend {
	backends := make([]*core.Backend, n)

	for i := 0; i < n; i++ {
		backends[i] = &core.Backend{
			Target: core.Target{
				Host: fmt.Sprintf("%s.%d", base, i+1),
				Port: fmt.Sprintf("%d", 1000+i),
			},
		}
	}

	return backends
}

// Prepare random list of clients
func prepareClients(n int) []DummyContext {

	clients := make([]DummyContext, n)

	for i := 0; i < n; i++ {

		ip := make([]byte, 4)
		rand.Read(ip)

		clients[i] = DummyContext{
			ip: ip,
		}
	}

	return clients

}

//TODO enable test when real consisten hashing will be implemented
/*
func TestIPHash2AddingBackendsRedistribution(t *testing.T) {
	rand.Seed(time.Now().Unix())
	balancer := &balance.Iphash2Balancer{}

	N := 50   // initial number of backends
	M := 1    // added number of backends
	C := 1000 // number of clients

	backends := prepareBackends("127.0.0", N)
	clients := prepareClients(C)

	// Perform balancing for on a given balancer, for clients versus backends
	d1, err := makeDistribution(balancer, backends, clients)
	if err != nil {
		t.Error(err)
		t.Fail()
	}

	extendedBackends := append(backends, prepareBackends("192.168.1", M)...)

	// Perform balancing for on a given balancer, for clients versus extended list of backends
	d2, err := makeDistribution(balancer, extendedBackends, clients)
	if err != nil {
		t.Error(err)
		t.Fail()
	}

	Q := 0 // number of rehashed clients

	// Q should not be bigger than C/ M+N

	// values should differ
	for k, v1 := range d1 {
		v2 := d2[k]
		if v1 != v2 {
			Q++
		}

	}

	if Q > C/(M+N) {
		t.Fail()
	}

}
*/

func TestIPHash1RemovingBackendsStability(t *testing.T) {

	balancer := &balance.Iphash1Balancer{}

	backends := prepareBackends("127.0.0", 4)
	clients := prepareClients(100)

	// Perform balancing for on a given balancer, for clients versus backends
	d1, err := makeDistribution(balancer, backends, clients)
	if err != nil {
		t.Error(err)
		t.Fail()
	}

	// Remove a backend from a list(second one)
	removedBackend := backends[1]
	backends = append(backends[:1], backends[2:]...)

	// Perform balancing on the same balancer, same clients, but backends missing one.
	d2, err := makeDistribution(balancer, backends, clients)
	if err != nil {
		t.Error(err)
		t.Fail()
	}

	// check the results
	for k, v1 := range d1 {

		// in the second try (d2) removed backend will be obviously changed to something else,
		// skipping it
		if v1 == removedBackend {
			continue
		}

		v2 := d2[k]

		// the second try (d2) should not have other changes, so that if some backend (not removed) was
		// elected previously, it should be elected now
		if v1 != v2 {
			t.Fail()
			break
		}

	}

}
