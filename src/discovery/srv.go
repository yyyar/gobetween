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
	srvUdpSize            = 4096
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
	c := dns.Client{Net: "udp", Timeout: timeout}
	m := dns.Msg{}

	m.SetQuestion(cfg.SrvLookupPattern, dns.TypeSRV)
	m.SetEdns0(srvUdpSize, true)
	r, _, err := c.Exchange(&m, cfg.SrvLookupServer)

	if err != nil {
		return nil, err
	}

	if len(r.Answer) == 0 {
		log.Warn("Empty response from", cfg.SrvLookupServer, cfg.SrvLookupPattern)
		return &[]core.Backend{}, nil
	}

	// Get hosts from A section
	hosts := make(map[string]string)
	for _, ans := range r.Extra {
		record := ans.(*dns.A)
		hosts[record.Header().Name] = record.A.String()
	}

	// Results for combined SRV + A
	results := []core.Backend{}
	for _, ans := range r.Answer {
		record := ans.(*dns.SRV)
		results = append(results, core.Backend{
			Target: core.Target{
				Host: hosts[record.Target],
				Port: fmt.Sprintf("%v", record.Port),
			},
			Priority: int(record.Priority),
			Weight:   int(record.Weight),
			Live:     true,
		})
	}

	return &results, nil
}
