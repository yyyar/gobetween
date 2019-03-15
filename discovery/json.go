package discovery

/**
 * docker.go - Docker API discovery implementation
 *
 * @author Ievgen Ponomarenko <kikomdev@gmail.com>
 * @author Yaroslav Pogrebnyak <yyyaroslav@gmail.com>
 */

import (
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"
	"time"

	"github.com/elgs/gojq"
	"github.com/yyyar/gobetween/config"
	"github.com/yyyar/gobetween/core"
	"github.com/yyyar/gobetween/logging"
	"github.com/yyyar/gobetween/utils"
)

const (
	jsonRetryWaitDuration      = 2 * time.Second
	jsonDefaultHttpTimeout     = 5 * time.Second
	jsonDefaultHostPattern     = "host"
	jsonDefaultPortPattern     = "port"
	jsonDefaultWeightPattern   = "weight"
	jsonDefaultPriorityPattern = "priority"
	jsonDefaultSniPattern      = "sni"
)

/**
 * Create new Discovery with Json fetch func
 */
func NewJsonDiscovery(cfg config.DiscoveryConfig) interface{} {

	/* replace with defaults if needed */

	if cfg.JsonHostPattern == "" {
		cfg.JsonHostPattern = jsonDefaultHostPattern
	}

	if cfg.JsonPortPattern == "" {
		cfg.JsonPortPattern = jsonDefaultPortPattern
	}

	if cfg.JsonWeightPattern == "" {
		cfg.JsonWeightPattern = jsonDefaultWeightPattern
	}

	if cfg.JsonPriorityPattern == "" {
		cfg.JsonPriorityPattern = jsonDefaultPriorityPattern
	}

	if cfg.JsonSniPattern == "" {
		cfg.JsonSniPattern = jsonDefaultSniPattern
	}

	d := Discovery{
		opts:  DiscoveryOpts{jsonRetryWaitDuration},
		fetch: jsonFetch,
		cfg:   cfg,
	}

	return &d
}

/**
 * Fetch / refresh backends from URL with json in response
 */
func jsonFetch(cfg config.DiscoveryConfig) (*[]core.Backend, error) {

	log := logging.For("jsonFetch")

	log.Info("fetching ", cfg.JsonEndpoint)

	// Make request
	timeout := utils.ParseDurationOrDefault(cfg.Timeout, jsonDefaultHttpTimeout)
	client := http.Client{Timeout: timeout}
	res, err := client.Get(cfg.JsonEndpoint)
	if err != nil {
		return nil, err
	}

	defer res.Body.Close()

	// Read response
	content, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}

	// Build query
	parsed, err := gojq.NewStringQuery(string(content))
	if err != nil {
		return nil, err
	}

	// parse query to array to ensure right format and get length of it
	parsedArray, err := parsed.QueryToArray(".")
	if err != nil {
		return nil, errors.New("Unexpected json in response")
	}

	var backends []core.Backend

	for k := range parsedArray {

		var key = "[" + strconv.Itoa(k) + "]."

		backend := core.Backend{
			Weight:   1,
			Priority: 1,
			Stats: core.BackendStats{
				Live: true,
			},
		}

		if backend.Host, err = parsed.QueryToString(key + cfg.JsonHostPattern); err != nil {
			return nil, err
		}

		// workaround to allow string or number port value
		port, err := parsed.Query(key + cfg.JsonPortPattern)
		if err != nil {
			return nil, err
		}

		// convert port to string (if not)
		backend.Port = fmt.Sprintf("%v", port)

		if weight, err := parsed.QueryToInt64(key + cfg.JsonWeightPattern); err == nil {
			backend.Weight = int(weight)
		}

		if priority, err := parsed.QueryToFloat64(key + cfg.JsonPriorityPattern); err == nil {
			backend.Priority = int(priority)
		}

		if sni, err := parsed.QueryToString(key + cfg.JsonSniPattern); err == nil {
			backend.Sni = sni
		}

		backends = append(backends, backend)
	}

	log.Info(backends)

	return &backends, nil
}
