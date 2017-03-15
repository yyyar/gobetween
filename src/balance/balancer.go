/**
 * balancer.go - balancer interface
 *
 * @author Yaroslav Pogrebnyak <yyyaroslav@gmail.com>
 */

package balance

import (
	"errors"
	"reflect"
	"regexp"
	"strings"

	"../config"
	"../core"
)

/**
 * Type registry of available Balancers
 */
var typeRegistry = make(map[string]reflect.Type)

/**
 * Initialize type registry
 */
func init() {
	typeRegistry["leastconn"] = reflect.TypeOf(LeastconnBalancer{})
	typeRegistry["roundrobin"] = reflect.TypeOf(RoundrobinBalancer{})
	typeRegistry["weight"] = reflect.TypeOf(WeightBalancer{})
	typeRegistry["iphash"] = reflect.TypeOf(IphashBalancer{})
	typeRegistry["leastbandwidth"] = reflect.TypeOf(LeastbandwidthBalancer{})
}

/**
 * Balancer interface
 */
type Balancer interface {

	/**
	 * Elect backend based on Balancer implementation
	 */
	Elect(core.Context, []*core.Backend) (*core.Backend, error)
}

type baseBalancer struct {
	sniConf  *config.Sni
	delegate Balancer
}

func (b *baseBalancer) compareSni(requestedSni string, backendSni string) (bool, error) {

	sniMatching := b.sniConf.HostnameMatchingStrategy

	switch sniMatching {
	case "regexp":
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

func (b *baseBalancer) Elect(ctx core.Context, backends []*core.Backend) (*core.Backend, error) {

	if b.sniConf == nil {
		return b.delegate.Elect(ctx, backends)
	}

	sni := ctx.Sni()
	strategy := b.sniConf.UnexpectedHostnameStrategy

	if sni == "" && strategy == "reject" {
		return nil, errors.New("Rejecting client due to an empty sni")
	}

	if sni == "" && strategy == "any" {
		return b.delegate.Elect(ctx, backends)
	}

	var filtered []*core.Backend

	for _, backend := range backends {

		match, err := b.compareSni(sni, backend.Sni)

		if err != nil {
			return nil, err
		}

		if match {
			filtered = append(filtered, backend)
		}

	}

	if len(filtered) > 0 {
		return b.delegate.Elect(ctx, filtered)
	}

	if strategy == "any" {
		return b.delegate.Elect(ctx, backends)
	}

	return nil, errors.New("Rejecting client due to not matching sni")

}

/**
 * Create new Balancer based on balancing strategy
 */
func New(sniConf *config.Sni, balance string) Balancer {
	return &baseBalancer{
		sniConf:  sniConf,
		delegate: reflect.New(typeRegistry[balance]).Elem().Addr().Interface().(Balancer),
	}
}
