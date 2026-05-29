package test

import (
	"testing"

	"github.com/yyyar/gobetween/balance"
	"github.com/yyyar/gobetween/balance/middleware"
	"github.com/yyyar/gobetween/core"
)

func TestMaxConnectionsMiddleware(t *testing.T) {
	// Create a simple round-robin balancer
	balancer := &balance.RoundrobinBalancer{}

	// Wrap it with our MaxConnectionsBalancer middleware
	maxConnBalancer := &middleware.MaxConnectionsMiddleware{
		Delegate: balancer,
	}

	// Create a test context
	context := DummyContext{}

	// Create test backends with different max_connections settings
	backends := []*core.Backend{
		{
			Target: core.Target{
				Host: "1",
				Port: "1",
			},
			MaxConnections: 10,
			Stats: core.BackendStats{
				ActiveConnections: 5, // Under limit
			},
		},
		{
			Target: core.Target{
				Host: "2",
				Port: "2",
			},
			MaxConnections: 10,
			Stats: core.BackendStats{
				ActiveConnections: 10, // At limit, should be excluded
			},
		},
		{
			Target: core.Target{
				Host: "3",
				Port: "3",
			},
			MaxConnections: 10,
			Stats: core.BackendStats{
				ActiveConnections: 15, // Over limit, should be excluded
			},
		},
		{
			Target: core.Target{
				Host: "4",
				Port: "4",
			},
			MaxConnections: 0, // No limit set
			Stats: core.BackendStats{
				ActiveConnections: 100, // High number of connections, but no limit
			},
		},
	}

	// Test 1: Only backends under their limit or with no limit should be elected
	selected := make(map[string]bool)

	// Run multiple elections to make sure our middleware is filtering correctly
	for i := 0; i < 100; i++ {
		backend, err := maxConnBalancer.Elect(context, backends)
		if err != nil {
			t.Fatal(err)
		}

		// Add to our list of selected backends
		selected[backend.Target.Host] = true

		// Verify the backend is eligible (either under limit or no limit)
		if backend.MaxConnections > 0 && backend.Stats.ActiveConnections >= uint(backend.MaxConnections) {
			t.Errorf("Backend %s elected despite exceeding max_connections (%d >= %d)",
				backend.Target.Host, backend.Stats.ActiveConnections, backend.MaxConnections)
		}
	}

	// Verify that only backends 1 and 4 were selected
	if !selected["1"] {
		t.Error("Backend 1 should have been selected (under limit)")
	}
	if selected["2"] {
		t.Error("Backend 2 should NOT have been selected (at limit)")
	}
	if selected["3"] {
		t.Error("Backend 3 should NOT have been selected (over limit)")
	}
	if !selected["4"] {
		t.Error("Backend 4 should have been selected (no limit)")
	}

	// Test 2: When all backends exceed their limits, an error should be returned
	allLimitedBackends := []*core.Backend{
		{
			Target: core.Target{
				Host: "1",
				Port: "1",
			},
			MaxConnections: 10,
			Stats: core.BackendStats{
				ActiveConnections: 10, // At limit
			},
		},
		{
			Target: core.Target{
				Host: "2",
				Port: "2",
			},
			MaxConnections: 5,
			Stats: core.BackendStats{
				ActiveConnections: 10, // Over limit
			},
		},
	}

	_, err := maxConnBalancer.Elect(context, allLimitedBackends)
	if err == nil {
		t.Error("Expected error when all backends exceed max_connections, but got none")
	}
}

