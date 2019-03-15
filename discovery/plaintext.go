package discovery

/**
 * plaintext.go - Plaintext discovery implementation
 *
 * @author Ievgen Ponomarenko <kikomdev@gmail.com>
 * @author Yaroslav Pogrebnyak <yyyaroslav@gmail.com>
 */

import (
	"io/ioutil"
	"net/http"
	"strings"
	"time"

	"github.com/yyyar/gobetween/config"
	"github.com/yyyar/gobetween/core"
	"github.com/yyyar/gobetween/logging"
	"github.com/yyyar/gobetween/utils"
	"github.com/yyyar/gobetween/utils/parsers"
)

const (
	plaintextDefaultRetryWaitDuration = 2 * time.Second
	plaintextDefaultHttpTimeout       = 5 * time.Second
)

/**
 * Create new Discovery with Plaintext fetch func
 */
func NewPlaintextDiscovery(cfg config.DiscoveryConfig) interface{} {

	if cfg.PlaintextRegexpPattern == "" {
		cfg.PlaintextRegexpPattern = parsers.DEFAULT_BACKEND_PATTERN
	}

	d := Discovery{
		opts:  DiscoveryOpts{plaintextDefaultRetryWaitDuration},
		fetch: plaintextFetch,
		cfg:   cfg,
	}

	return &d
}

/**
 * Fetch / refresh backends from URL with plain text
 */
func plaintextFetch(cfg config.DiscoveryConfig) (*[]core.Backend, error) {

	log := logging.For("plaintextFetch")

	log.Info("Fetching ", cfg.PlaintextEndpoint)

	// Make request
	timeout := utils.ParseDurationOrDefault(cfg.Timeout, plaintextDefaultHttpTimeout)
	client := http.Client{Timeout: timeout}
	res, err := client.Get(cfg.PlaintextEndpoint)
	if err != nil {
		return nil, err
	}

	defer res.Body.Close()

	// Read response
	content, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}

	backends := []core.Backend{}
	lines := strings.Split(string(content), "\n")

	// Iterate and parse
	for _, line := range lines {

		if line == "" {
			continue
		}

		backend, err := parsers.ParseBackend(line, cfg.PlaintextRegexpPattern)
		if err != nil {
			log.Warn("Cant parse ", line, err)
			continue
		}

		backends = append(backends, *backend)
	}

	return &backends, err
}
