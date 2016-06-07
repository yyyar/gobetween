/**
 * balancer.go - balancer interface
 *
 * @author Yaroslav Pogrebnyak <yyyaroslav@gmail.com>
 */

package balance

import (
	"../core"
	"reflect"
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
}

/**
 * Create new Balancer based on balancing strategy
 */
func New(strategy string) Balancer {
	return reflect.New(typeRegistry[strategy]).Elem().Addr().Interface().(Balancer)
}

/**
 * Balancer interface
 */
type Balancer interface {

	/**
	 * Elect backend based on Balancer implementation
	 */
	Elect(*core.Context, []core.Backend) (*core.Backend, error)
}
