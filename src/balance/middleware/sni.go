package middleware

/**
 * sni.go - sni middleware
 *
 * @author Illarion Kovalchuk <illarion.kovalchuk@gmail.com>
 * @author Yaroslav Pogrebnyak <yyyaroslav@gmail.com>
 */

import (
	"errors"
	"regexp"
	"strings"

	"github.com/yyyar/gobetween/config"
	"github.com/yyyar/gobetween/core"
	"github.com/yyyar/gobetween/logging"
)

/**
 * SniBalancer middleware delegate
 */
type SniBalancer struct {
	SniConf  *config.Sni
	Delegate core.Balancer
}

/**
 * Elect backend using sni pre-processing
 */
func (sniBalancer *SniBalancer) Elect(ctx core.Context, backends []*core.Backend) (*core.Backend, error) {

	/* ------ try find matching to requesedSni backends ------ */

	matchedBackends := sniBalancer.matchingBackends(ctx.Sni(), backends)
	if len(matchedBackends) > 0 {
		return sniBalancer.Delegate.Elect(ctx, matchedBackends)
	}

	/* ------ if no matched backends, fallback to unexpected hostname strategy ------ */

	switch sniBalancer.SniConf.UnexpectedHostnameStrategy {
	case "reject":
		return nil, errors.New("No matching sni [" + ctx.Sni() + "] found, rejecting due to 'reject' unexpected hostname strategy")

	case "any":
		return sniBalancer.Delegate.Elect(ctx, backends)

	default:
		if ctx.Sni() == "" {
			return sniBalancer.Delegate.Elect(ctx, []*core.Backend{})
		}

		// default, select only from backends without any sni
		return sniBalancer.Delegate.Elect(ctx, sniBalancer.matchingBackends("", backends))
	}
}

/**
 * Filter out backends that match requestedSni
 */
func (sniBalancer *SniBalancer) matchingBackends(requestedSni string, backends []*core.Backend) []*core.Backend {

	log := logging.For("balance/middleware/sni")

	var matchedBackends []*core.Backend

	for _, backend := range backends {

		match, err := sniBalancer.matchSni(requestedSni, backend.Sni)

		if err != nil {
			log.Error(err)
			continue
		}

		if match {
			matchedBackends = append(matchedBackends, backend)
		}
	}

	return matchedBackends
}

/**
 * Try match requested sni to actual backend sni
 */
func (sniBalancer *SniBalancer) matchSni(requestedSni string, backendSni string) (bool, error) {

	sniMatching := sniBalancer.SniConf.HostnameMatchingStrategy

	switch sniMatching {
	case "regexp":

		if backendSni == "" && requestedSni != "" {
			return false, nil
		}

		regexp, err := regexp.Compile(backendSni)
		if err != nil {
			return false, err
		}

		return regexp.MatchString(requestedSni), nil

	case "exact":
		return strings.ToLower(requestedSni) == strings.ToLower(backendSni), nil

	default:
		return false, errors.New("Unsupported sni matching mechanism: " + sniMatching)
	}

}
