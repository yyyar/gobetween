/**
 * registry.go - balancers registry
 *
 * @author Yaroslav Pogrebnyak <yyyaroslav@gmail.com>
 */

package balance

import (
	"reflect"

	"./middleware"

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
	typeRegistry["iphash1"] = reflect.TypeOf(Iphash1Balancer{})
	typeRegistry["leastbandwidth"] = reflect.TypeOf(LeastbandwidthBalancer{})
}

/**
 * Create new Balancer based on balancing strategy
 * Wrap it in middlewares if needed
 */
func New(sniConf *config.Sni, balance string) core.Balancer {
	balancer := reflect.New(typeRegistry[balance]).Elem().Addr().Interface().(core.Balancer)

	if sniConf == nil {
		return balancer
	}

	return &middleware.SniBalancer{
		SniConf:  sniConf,
		Delegate: balancer,
	}
}
