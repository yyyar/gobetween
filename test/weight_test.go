package test

import (
	"math"
	"math/rand"
	"testing"
	"time"

	"github.com/yyyar/gobetween/balance"
	"github.com/yyyar/gobetween/core"
)

func TestOnlyBestPriorityBackendsElected(t *testing.T) {
	rand.Seed(time.Now().Unix())
	balancer := &balance.WeightBalancer{}
	var context core.Context

	context = DummyContext{}

	backends := []*core.Backend{
		{
			Priority: 0,
			Weight:   0,
		},
		{
			Priority: 0,
			Weight:   1,
		},
		{
			Priority: 0,
			Weight:   2,
		},
		{
			Priority: 1,
			Weight:   0,
		},
		{
			Priority: 1,
			Weight:   1,
		},
		{
			Priority: 1,
			Weight:   2,
		},
		{
			Priority: 2,
			Weight:   0,
		},
		{
			Priority: 2,
			Weight:   1,
		},
		{
			Priority: 2,
			Weight:   2,
		},
	}

	hits := make(map[int]bool)

	for try := 0; try < 100; try++ {
		backend, err := balancer.Elect(context, backends)
		if err != nil {
			t.Fatal(err)
		}

		hits[backend.Priority] = true
		if len(hits) > 1 {
			t.Error("Backends with different priority elected")
		}

		if backend.Priority != 0 {
			t.Error("Backends with not optimal priority elected")
		}

	}

}

func TestAllWeightsEqualTo0Distribution(t *testing.T) {
	rand.Seed(time.Now().Unix())
	balancer := &balance.WeightBalancer{}
	var context core.Context

	context = DummyContext{}

	backends := []*core.Backend{
		{
			Target: core.Target{
				Host: "1",
				Port: "1",
			},
			Weight: 0,
		},
		{
			Target: core.Target{
				Host: "2",
				Port: "2",
			},
			Weight: 0,
		},
		{
			Target: core.Target{
				Host: "3",
				Port: "3",
			},
			Weight: 0,
		},
	}

	hits := make(map[string]bool)

	for try := 0; try < 100; try++ {
		backend, err := balancer.Elect(context, backends)
		if err != nil {
			t.Fatal(err)
		}

		hits[backend.Target.Host] = true
		if len(hits) == 3 {
			return
		}

	}

	if len(hits) != 3 {
		t.Error("Group of backends with weight = 0 has some backneds that are never elected")
	}
}

func TestWeightDistribution(t *testing.T) {
	rand.Seed(time.Now().Unix())
	balancer := &balance.WeightBalancer{}
	var context core.Context

	context = DummyContext{}

	backends := []*core.Backend{
		{
			Priority: 1,
			Weight:   20,
		},
		{
			Priority: 1,
			Weight:   15,
		},
		{
			Priority: 1,
			Weight:   25,
		},
		{
			Priority: 1,
			Weight:   40,
		},
		{
			// this backend is ignored
			Priority: 2,
			Weight:   244,
		},
	}

	//shuffle
	for s := 0; s < 100; s++ {
		i := rand.Intn(len(backends))
		j := rand.Intn(len(backends))
		if i == j {
			continue
		}
		backends[i], backends[j] = backends[j], backends[i]
	}

	quantity := make(map[int]int)

	for _, backend := range backends {
		if backend.Priority > 1 {
			continue
		}
		quantity[backend.Weight] = 0
	}

	n := 10000
	for try := 0; try < 100*n; try++ {
		backend, err := balancer.Elect(context, backends)
		if err != nil {
			t.Fatal(err)
		}

		quantity[backend.Weight] += 1
	}

	for k, v := range quantity {
		if math.Abs(float64(v)/float64(n)-float64(k)) > 0.5 {
			t.Error(k, ":", float64(v)/float64(n))
		}

	}
}
