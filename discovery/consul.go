package discovery

/**
 * consul.go - Consul API discovery implementation
 *
 * @author Yaroslav Pogrebnyak <yyyaroslav@gmail.com>
 */

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	consul "github.com/hashicorp/consul/api"
	"github.com/yyyar/gobetween/config"
	"github.com/yyyar/gobetween/core"
	"github.com/yyyar/gobetween/logging"
	"github.com/yyyar/gobetween/utils"
)

const (
	consulRetryWaitDuration = 2 * time.Second
	consulTimeout           = 2 * time.Second
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

	// Prepare vars for http client
	// TODO move http & consul client creation to constructor
	scheme := "http"
	transport := &http.Transport{
		DisableKeepAlives: true,
	}

	// Enable tls if needed
	if cfg.ConsulTlsEnabled {
		tlsConfig := &consul.TLSConfig{
			Address:  cfg.ConsulHost,
			CertFile: cfg.ConsulTlsCertPath,
			KeyFile:  cfg.ConsulTlsKeyPath,
			CAFile:   cfg.ConsulTlsCacertPath,
		}
		tlsClientConfig, err := consul.SetupTLSConfig(tlsConfig)
		if err != nil {
			return nil, err
		}
		transport.TLSClientConfig = tlsClientConfig
		scheme = "https"
	}

	// Parse http timeout
	timeout := utils.ParseDurationOrDefault(cfg.Timeout, consulTimeout)

	// Create consul client
	client, _ := consul.NewClient(&consul.Config{
		Scheme:     scheme,
		Address:    cfg.ConsulHost,
		Datacenter: cfg.ConsulDatacenter,
		HttpAuth: &consul.HttpBasicAuth{
			Username: cfg.ConsulAuthUsername,
			Password: cfg.ConsulAuthPassword,
		},
		HttpClient: &http.Client{Timeout: timeout, Transport: transport},
	})

	// Query service
	service, _, err := client.Health().Service(cfg.ConsulServiceName, cfg.ConsulServiceTag, cfg.ConsulServicePassingOnly, nil)
	if err != nil {
		return nil, err
	}

	// Gather backends
	backends := []core.Backend{}
	for _, entry := range service {
		s := entry.Service
		sni := ""

		for _, tag := range s.Tags {
			split := strings.SplitN(tag, "=", 2)

			if len(split) != 2 {
				continue
			}

			if split[0] != "sni" {
				continue
			}
			sni = split[1]
		}

		var host string
		if s.Address != "" {
			host = s.Address
		} else {
			host = entry.Node.Address
		}

		backends = append(backends, core.Backend{
			Target: core.Target{
				Host: host,
				Port: fmt.Sprintf("%v", s.Port),
			},
			Priority: 1,
			Weight:   1,
			Stats: core.BackendStats{
				Live: true,
			},
			Sni: sni,
		})
	}

	return &backends, nil
}
