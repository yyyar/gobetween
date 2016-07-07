/**
 * srv.go - SRV record DNS resolve discovery implementation
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
	"github.com/miekg/dns"
	"time"
)

const (
	srvRetryWaitDuration  = 2 * time.Second
	srvDefaultWaitTimeout = 5 * time.Second
)

func NewSrvDiscovery(cfg config.DiscoveryConfig) interface{} {

	d := Discovery{
		opts:  DiscoveryOpts{srvRetryWaitDuration},
		fetch: srvFetch,
		cfg:   cfg,
	}

	return &d
}

/**
 * Create new Discovery with Srv fetch func
 */
func srvFetch(cfg config.DiscoveryConfig) (*[]core.Backend, error) {

	log := logging.For("srvFetch")

	log.Info("Fetching ", cfg.SrvLookupServer, " ", cfg.SrvLookupPattern)

	timeout := utils.ParseDurationOrDefault(cfg.Timeout, srvDefaultWaitTimeout)
	c := dns.Client{Timeout: timeout}
	m := dns.Msg{}

	m.SetQuestion(cfg.SrvLookupPattern, dns.TypeSRV)
	r, _, err := c.Exchange(&m, cfg.SrvLookupServer)

	if err != nil {
		return nil, err
	}

	if len(r.Answer) == 0 {
		log.Warn("Empty response from", cfg.SrvLookupServer, cfg.SrvLookupPattern)
		return &[]core.Backend{}, nil
	}

	// Results for combined SRV + A results
	result := make(map[string]core.Backend)

	for _, ans := range r.Answer {
		record := ans.(*dns.SRV)
		result[record.Target] = core.Backend{
			Target: core.Target{
				Host: record.Target,
				Port: fmt.Sprintf("%v", record.Port),
			},
			Priority: int(record.Priority),
			Weight:   int(record.Weight),
			Stats: core.BackendStats{
				Live: true,
			},
		}
	}

	for _, ans := range r.Extra {
		record := ans.(*dns.A)
		b := result[record.Hdr.Name]
		b.Host = record.A.String()
		result[record.Hdr.Name] = b
	}

	// Make list of backends from results map
	var values []core.Backend
	for _, value := range result {
		values = append(values, value)
	}

	return &values, nil
}
