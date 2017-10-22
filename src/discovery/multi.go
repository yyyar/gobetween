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
)

var multiSubDiscoveries []*Discovery

var backends = make(map[int][]core.Backend)

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

		d := New(discoveryCfg.Kind, discoveryCfg)

		multiSubDiscoveries = append(multiSubDiscoveries, d)
		d.Start()
		go (func(i int) {
			for {
				b := <-d.out
				backends[i] = b
			}
		})(i)
	}

	return &d
}

/**
 * Start discovery
 */
func multiFetch(cfg config.DiscoveryConfig) (*[]core.Backend, error) {

	//	log := logging.For("discovery/multi")
	var result []core.Backend

	for _, bs := range backends {

		for _, bss := range bs {
			result = append(result, bss)
		}
	}

	return &result, nil
}
