package access

/**
 * access.go - access
 *
 * @author Yaroslav Pogrebnyak <yyyaroslav@gmail.com>
 */

import (
	"errors"
	"net"

	"github.com/yyyar/gobetween/config"
)

/**
 * Access defines access rules chain
 */
type Access struct {
	AllowDefault bool
	Rules        []AccessRule
}

/**
 * Creates new Access based on config
 */
func NewAccess(cfg *config.AccessConfig) (*Access, error) {

	if cfg == nil {
		return nil, errors.New("AccessConfig is nil")
	}

	if cfg.Default == "" {
		cfg.Default = "allow"
	}

	if cfg.Default != "allow" && cfg.Default != "deny" {
		return nil, errors.New("AccessConfig Unexpected Default: " + cfg.Default)
	}

	access := Access{
		AllowDefault: cfg.Default == "allow",
		Rules:        []AccessRule{},
	}

	// Parse rules
	for _, r := range cfg.Rules {
		rule, err := ParseAccessRule(r)
		if err != nil {
			return nil, err
		}
		access.Rules = append(access.Rules, *rule)
	}

	return &access, nil
}

/**
 * Checks if ip is allowed
 */
func (this *Access) Allows(ip *net.IP) bool {

	for _, r := range this.Rules {
		if r.Matches(ip) {
			return r.Allows()
		}
	}

	return this.AllowDefault
}
