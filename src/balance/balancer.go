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
	cfg      config.Server
	delegate Balancer
}

func compareSni(requestedSni string, backendSni string) bool {
	if regexp, err := regexp.Compile(backendSni); err == nil {
		return regexp.MatchString(requestedSni)
	}

	r := strings.ToLower(requestedSni)
	b := strings.ToLower(backendSni)

	return r == b
}

func (b *baseBalancer) Elect(ctx core.Context, backends []*core.Backend) (*core.Backend, error) {

	if !b.cfg.Sni.Enabled {
		return b.delegate.Elect(ctx, backends)
	}

	sni := ctx.Sni()
	strategy := b.cfg.Sni.UnexpectedHostnameStrategy

	if sni == "" && strategy == "reject" {
		return nil, errors.New("Rejecting client due to an empty sni")
	}

	if sni == "" && strategy == "any" {
		return b.delegate.Elect(ctx, backends)
	}

	var filtered []*core.Backend

	for _, b := range backends {

		if compareSni(sni, b.Sni) {
			filtered = append(filtered, b)
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
func New(cfg config.Server) Balancer {
	return &baseBalancer{
		cfg:      cfg,
		delegate: reflect.New(typeRegistry[cfg.Balance]).Elem().Addr().Interface().(Balancer),
	}
}
