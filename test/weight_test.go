package test

import (
	"math"
	"math/rand"
	"testing"
	"time"

	"../src/balance"
	"../src/core"
)

func TestWeightDistribution(t *testing.T) {
	rand.Seed(time.Now().Unix())
	balancer := &balance.WeightBalancer{}
	var context core.Context

	context = DummyContext{}

	backends := []*core.Backend{
		{
			Weight: 20,
		},
		{
			Weight: 15,
		},
		{
			Weight: 25,
		},
		{
			Weight: 40,
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
