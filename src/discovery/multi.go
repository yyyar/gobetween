/**
 * multi.go - multi discovery implementation
 *
 * @author Yaroslav Pogrebnyak <yyyaroslav@gmail.com>
 */

package discovery

import (
	"../config"
	"../core"
	"../logging"
	"sync"
)

var backends = make(map[int][]core.Backend)
var mu = sync.RWMutex{}

/**
 * Creates new static discovery
 */
func NewMultiDiscovery(cfg config.DiscoveryConfig) interface{} {

	log := logging.For("discovery/multi")

	d := Discovery{
		opts:  DiscoveryOpts{0},
		cfg:   cfg,
		fetch: multiFetch,
	}

	for i, discoveryCfg := range cfg.Multi {

		if discoveryCfg.Kind == "multi" {
			log.Warn("Can't have multi discovry inside multi discovery. Ignoring it...")
			continue
		}

		if discoveryCfg.Failpolicy == "" {
			discoveryCfg.Failpolicy = cfg.Failpolicy
		}

		if discoveryCfg.Interval == "" {
			discoveryCfg.Interval = cfg.Interval
		}

		if discoveryCfg.Timeout == "" {
			discoveryCfg.Timeout = cfg.Timeout
		}

		subDiscovery := New(discoveryCfg.Kind, discoveryCfg)
		subDiscovery.Start()

		go func(i int) {
			c := subDiscovery.Discover()
			for {
				b := <-c
				mu.Lock()
				backends[i] = b
				mu.Unlock()
			}
		}(i)
	}

	return &d
}

/**
 * Start discovery
 */
func multiFetch(cfg config.DiscoveryConfig) (*[]core.Backend, error) {
	//	log := logging.For("discovery/multi")
	var result []core.Backend

	mu.RLock()
	for _, bs := range backends {
		for _, bss := range bs {
			result = append(result, bss)
		}
	}
	mu.RUnlock()

	return &result, nil
}
