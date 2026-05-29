package access

/**
 * rule.go - access rule
 *
 * @author Yaroslav Pogrebnyak <yyyaroslav@gmail.com>
 */

import (
	"errors"
	"net"
	"strings"
)

/**
 * AccessRule defines order (access, deny)
 * and IP or Network
 */
type AccessRule struct {
	Allow     bool
	IsNetwork bool
	Ip        *net.IP
	Network   *net.IPNet
}

/**
 * Parses string to AccessRule
 */
func ParseAccessRule(rule string) (*AccessRule, error) {

	parts := strings.Split(rule, " ")
	if len(parts) != 2 {
		return nil, errors.New("Bad access rule format: " + rule)
	}

	r := parts[0]
	cidrOrIp := parts[1]

	if r != "allow" && r != "deny" {
		return nil, errors.New("Cant parse rule definition " + rule)
	}

	// try check if cidrOrIp is ip and handle

	ipShould := net.ParseIP(cidrOrIp)
	if ipShould != nil {
		return &AccessRule{
			Allow:     r == "allow",
			Ip:        &ipShould,
			IsNetwork: false,
			Network:   nil,
		}, nil
	}

	_, ipNetShould, _ := net.ParseCIDR(cidrOrIp)
	if ipNetShould != nil {
		return &AccessRule{
			Allow:     r == "allow",
			Ip:        nil,
			IsNetwork: true,
			Network:   ipNetShould,
		}, nil
	}

	return nil, errors.New("Cant parse acces rule target, not an ip or cidr: " + cidrOrIp)

}

/**
 * Checks if ip matches access rule
 */
func (this *AccessRule) Matches(ip *net.IP) bool {

	switch this.IsNetwork {
	case true:
		return this.Network.Contains(*ip)
	case false:
		return (*this.Ip).Equal(*ip)
	}

	return false
}

/**
 * Checks is it's allow or deny rule
 */
func (this *AccessRule) Allows() bool {
	return this.Allow
}
