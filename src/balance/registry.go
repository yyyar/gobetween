package balance

/**
 * registry.go - balancers registry
 *
 * @author Yaroslav Pogrebnyak <yyyaroslav@gmail.com>
 */

import (
	"reflect"

	"github.com/yyyar/gobetween/balance/middleware"
	"github.com/yyyar/gobetween/config"
	"github.com/yyyar/gobetween/core"
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
	typeRegistry["iphash1"] = reflect.TypeOf(Iphash1Balancer{})
	typeRegistry["leastbandwidth"] = reflect.TypeOf(LeastbandwidthBalancer{})
}

/**
 * Create new Balancer based on balancing strategy
 * Wrap it in middlewares if needed
 */
func New(sniConf *config.Sni, balance string) core.Balancer {

	// Create the base balancer
	balancer := reflect.New(typeRegistry[balance]).Elem().Addr().Interface().(core.Balancer)

	// Apply max connections middleware (always applied)
	balancer = &middleware.MaxConnectionsMiddleware{
		Delegate: balancer,
	}

	// Apply SNI middleware if configured
	if sniConf != nil {
		balancer = &middleware.SniMiddleware{
			SniConf:  sniConf,
			Delegate: balancer,
		}
	}

	return balancer
}
