package discovery

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
	"time"

	"../config"
	"../core"
	"../logging"
	"../utils"
)

const (
	GobetweenRetryWaitDuration  = 2 * time.Second
	GobetweenDefaultHttpTimeout = 5 * time.Second
)

/**
 * Create a new Discovery with gobetween fetch func
 */
func NewGobetweenDiscovery(cfg config.DiscoveryConfig) interface{} {

	d := Discovery{
		opts:  DiscoveryOpts{GobetweenRetryWaitDuration},
		fetch: gobetweenFetch,
		cfg:   cfg,
	}

	return &d
}

/**
 * Fetch / refresh backends from gobetween server
 */
func gobetweenFetch(cfg config.DiscoveryConfig) (*[]core.Backend, error) {

	// Create backends for all API servers
	var backends []core.Backend

	for serverName, apiConfig := range cfg.GobetweenAPIServers {
		apiServerBackend, err := gobetweenQueryAPIServer(cfg, *apiConfig, serverName)
		if err != nil {
			return nil, err
		}

		backends = append(backends, *apiServerBackend)
	}

	return &backends, nil
}

func gobetweenQueryAPIServer(cfg config.DiscoveryConfig, apiConfig config.GobetweenDiscoveryAPIConfig, serverName string) (*core.Backend, error) {
	logLabel := fmt.Sprintf("gobetweenAPIFetch %s", apiConfig.APIAddress)
	log := logging.For(logLabel)

	// Make request
	apiEndpoint := apiConfig.APIAddress + "/servers"

	timeout := utils.ParseDurationOrDefault(cfg.Timeout, GobetweenDefaultHttpTimeout)
	client := http.Client{Timeout: timeout}
	req, err := http.NewRequest("GET", apiEndpoint, nil)
	if err != nil {
		return nil, err
	}

	if apiConfig.APIUsername != "" && apiConfig.APIPassword != "" {
		req.SetBasicAuth(apiConfig.APIUsername, apiConfig.APIPassword)
	}

	res, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	// Read response
	content, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}

	// Convert JSON response to interface
	var servers interface{}
	err = json.Unmarshal(content, &servers)
	if err != nil {
		return nil, err
	}

	// Begin parsing JSON, loop through servers
	serversMap, ok := servers.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("Unable to parse gobetween /servers")
	}

	serverInfo, ok := serversMap[serverName]
	if !ok {
		log.Debugf("Gobetween server %s did not contain server %s",
			apiConfig.APIAddress, serverName)
		return nil, nil
	}

	serverInfoMap, ok := serverInfo.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("Unable to parse %s information", serverName)
	}

	// Determine the downstream address and port
	var bindPort string
	var bindHost string

	// Use the API address as the default Host
	url, err := url.Parse(apiConfig.APIAddress)
	if err != nil {
		return nil, fmt.Errorf("Unable to parse Gobetween API address: %s", err)
	}
	hostParts := strings.Split(url.Host, ":")
	switch len(hostParts) {
	case 1:
		bindHost = url.Host
	case 2:
		bindHost = hostParts[0]
	}

	// Parse the discovered server's "bind" property.
	// If it's in the form of host:port, use those for the backend.
	// If it's only :port, use the API address.
	bindParts := strings.Split(serverInfoMap["bind"].(string), ":")
	if len(bindParts) == 2 {
		if bindParts[0] != "" {
			bindHost = bindParts[0]
		}

		bindPort = bindParts[1]
	}

	// Build the backend
	backend := core.Backend{
		Target: core.Target{
			Host: bindHost,
			Port: bindPort,
		},
		Weight:   apiConfig.BackendWeight,
		Priority: apiConfig.BackendPriority,
		Stats: core.BackendStats{
			Live: true,
		},
	}

	if sni, ok := serverInfoMap["sni"].(string); ok {
		backend.Sni = sni
	}

	log.Info(&backend)

	return &backend, nil
}
