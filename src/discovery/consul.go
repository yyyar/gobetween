/**
 * consul.go - Consul API discovery implementation
 *
 * @author Yaroslav Pogrebnyak <yyyaroslav@gmail.com>
 */

package discovery

import (
	"../config"
	"../core"
	"../logging"
	"fmt"
	consul "github.com/hashicorp/consul/api"
	"time"
)

const (
	consulRetryWaitDuration = 2 * time.Second
)

/**
 * Create new Discovery with Consul fetch func
 */
func NewConsulDiscovery(cfg config.DiscoveryConfig) interface{} {

	d := Discovery{
		opts:  DiscoveryOpts{consulRetryWaitDuration},
		fetch: consulFetch,
		cfg:   cfg,
	}

	return &d
}

/**
 * Fetch backends from Consul API
 */
func consulFetch(cfg config.DiscoveryConfig) (*[]core.Backend, error) {

	log := logging.For("consulFetch")

	log.Info("Fetching ", cfg)

	c, _ := consul.NewClient(&consul.Config{
		Address: cfg.ConsulHost,
	})

	service, _, err := c.Health().Service(cfg.ConsulServiceName, cfg.ConsulServiceTag, cfg.ConsulServicePassingOnly, nil)

	if err != nil {
		return nil, err
	}

	backends := []core.Backend{}
	for _, entry := range service {
		s := entry.Service
		backends = append(backends, core.Backend{
			Target: core.Target{
				Host: s.Address,
				Port: fmt.Sprintf("%v", s.Port),
			},
			Priority: 1,
			Weight:   1,
			Stats: core.BackendStats{
				Live: true,
			},
		})
	}

	return &backends, nil
}
