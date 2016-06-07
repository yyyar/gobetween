/**
 * docker.go - Docker API discovery implementation
 *
 * @author Yaroslav Pogrebnyak <yyyaroslav@gmail.com>
 */

package discovery

import (
	"../config"
	"../core"
	"../logging"
	"../utils"
	"fmt"
	"github.com/fsouza/go-dockerclient"
	"time"
)

const (
	dockerRetryWaitDuration = 2 * time.Second
	dockerTimeout           = 5 * time.Second
)

/**
 * Create new Discovery with Docker fetch func
 */
func NewDockerDiscovery(cfg config.DiscoveryConfig) interface{} {

	d := Discovery{
		opts:  DiscoveryOpts{dockerRetryWaitDuration},
		fetch: dockerFetch,
		cfg:   cfg,
	}

	return &d
}

/**
 * Fetch backends from Docker API
 */
func dockerFetch(cfg config.DiscoveryConfig) (*[]core.Backend, error) {

	log := logging.For("dockerFetch")

	log.Info("Fetching ", cfg.DockerEndpoint, " ", cfg.DockerContainerLabel, " ", cfg.DockerContainerPrivatePort)

	/* Creare docke client */
	client, err := docker.NewClient(cfg.DockerEndpoint)
	if err != nil {
		return nil, err
	}

	/* Set timeout */
	client.HTTPClient.Timeout = utils.ParseDurationOrDefault(cfg.Timeout, dockerTimeout)

	/* Add filter labels if any */
	var filters map[string][]string
	if cfg.DockerContainerLabel != "" {
		filters = map[string][]string{"label": []string{cfg.DockerContainerLabel}}
	}

	/* Fetch containers */
	containers, err := client.ListContainers(docker.ListContainersOptions{Filters: filters})
	if err != nil {
		return nil, err
	}

	/* Create backends from response */

	backends := []core.Backend{}

	for _, container := range containers {
		for _, port := range container.Ports {

			if port.PrivatePort != cfg.DockerContainerPrivatePort {
				continue
			}

			backends = append(backends, core.Backend{
				Target: core.Target{
					Host: port.IP,
					Port: fmt.Sprintf("%v", port.PublicPort),
				},
				Priority: 1,
				Weight:   1,
				Live:     true,
			})
		}
	}

	return &backends, nil
}
