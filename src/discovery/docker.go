package discovery

/**
 * docker.go - Docker API discovery implementation
 *
 * @author Yaroslav Pogrebnyak <yyyaroslav@gmail.com>
 */

import (
	"errors"
	"fmt"
	"regexp"
	"time"

	docker "github.com/fsouza/go-dockerclient"
	"github.com/yyyar/gobetween/config"
	"github.com/yyyar/gobetween/core"
	"github.com/yyyar/gobetween/logging"
	"github.com/yyyar/gobetween/utils"
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

	var client *docker.Client
	var err error

	if cfg.DockerTlsEnabled {

		// Client cert and key files should be specified together (or both not specified)
		// Ca cert may be not specified, so not checked here
		if (cfg.DockerTlsCertPath == "") != (cfg.DockerTlsKeyPath == "") {
			return nil, errors.New("Missing key or certificate required for TLS client validation")
		}

		client, err = docker.NewTLSClient(cfg.DockerEndpoint, cfg.DockerTlsCertPath, cfg.DockerTlsKeyPath, cfg.DockerTlsCacertPath)

	} else {
		client, err = docker.NewClient(cfg.DockerEndpoint)
	}

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

			containerHost := dockerDetermineContainerHost(client, container.ID, cfg, port.IP)

			backends = append(backends, core.Backend{
				Target: core.Target{
					Host: containerHost,
					Port: fmt.Sprintf("%v", port.PublicPort),
				},
				Priority: 1,
				Weight:   1,
				Stats: core.BackendStats{
					Live: true,
				},
				Sni: container.Labels["sni"],
			})
		}
	}

	return &backends, nil
}

/**
 * Determines container host
 */
func dockerDetermineContainerHost(client *docker.Client, id string, cfg config.DiscoveryConfig, portHost string) string {

	log := logging.For("dockerDetermineContainerHost")

	/* If host env var specified, try to get it from container vars */

	if cfg.DockerContainerHostEnvVar != "" {

		container, err := client.InspectContainer(id)

		if err != nil {
			log.Warn(err)
		} else {
			var e docker.Env = container.Config.Env
			h := e.Get(cfg.DockerContainerHostEnvVar)
			if h != "" {
				return h
			}
		}
	}

	/* If container portHost is not 'all interfaces', return it since it's good enough */

	if portHost != "0.0.0.0" {
		return portHost
	}

	/* Last chance, try to parse docker host from endpoint string */

	var reg = regexp.MustCompile("(.*?)://(?P<host>[-.A-Za-z0-9]+)/?(.*)")
	match := reg.FindStringSubmatch(cfg.DockerEndpoint)

	if len(match) == 0 {
		return portHost
	}

	result := make(map[string]string)

	// get named capturing groups
	for i, name := range reg.SubexpNames() {
		if name != "" {
			result[name] = match[i]
		}
	}

	h, ok := result["host"]
	if !ok {
		return portHost
	}

	return h
}
