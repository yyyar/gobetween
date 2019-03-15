package discovery

/**
 * exec.go - Exec external process discovery implementation
 *
 * @author Yaroslav Pogrebnyak <yyyaroslav@gmail.com>
 * @author Ievgen Ponomarenko <kikomdev@gmail.com>
 */

import (
	"strings"
	"time"

	"github.com/yyyar/gobetween/config"
	"github.com/yyyar/gobetween/core"
	"github.com/yyyar/gobetween/logging"
	"github.com/yyyar/gobetween/utils"
	"github.com/yyyar/gobetween/utils/parsers"
)

const (
	execRetryWaitDuration   = 2 * time.Second
	execResponseWaitTimeout = 3 * time.Second
)

/**
 * Create new Discovery with Exec fetch func
 */
func NewExecDiscovery(cfg config.DiscoveryConfig) interface{} {

	d := Discovery{
		opts:  DiscoveryOpts{execRetryWaitDuration},
		fetch: execFetch,
		cfg:   cfg,
	}

	return &d
}

/**
 * Fetch / refresh backends exec process
 */
func execFetch(cfg config.DiscoveryConfig) (*[]core.Backend, error) {

	log := logging.For("execFetch")

	log.Info("Fetching ", cfg.ExecCommand)

	timeout := utils.ParseDurationOrDefault(cfg.Timeout, execResponseWaitTimeout)
	out, err := utils.ExecTimeout(timeout, cfg.ExecCommand...)
	if err != nil {
		return nil, err
	}

	backends := []core.Backend{}

	for _, line := range strings.Split(string(out), "\n") {

		if line == "" {
			continue
		}

		backend, err := parsers.ParseBackendDefault(line)
		if err != nil {
			log.Warn(err)
			continue
		}

		backends = append(backends, *backend)
	}

	log.Info("Fetched ", backends)

	return &backends, nil
}
