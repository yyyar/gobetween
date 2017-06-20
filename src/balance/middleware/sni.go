package middleware

import (
	"errors"
	"regexp"
	"strings"

	"../../config"
	"../../core"
)

type SniBalancer struct {
	SniConf  *config.Sni
	Delegate core.Balancer
}

func (b *SniBalancer) compareSni(requestedSni string, backendSni string) (bool, error) {

	sniMatching := b.SniConf.HostnameMatchingStrategy

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

func (b *SniBalancer) Elect(ctx core.Context, backends []*core.Backend) (*core.Backend, error) {

	sni := ctx.Sni()
	strategy := b.SniConf.UnexpectedHostnameStrategy

	if sni == "" && strategy == "reject" {
		return nil, errors.New("Rejecting client due to an empty sni")
	}

	if sni == "" && strategy == "any" {
		return b.Delegate.Elect(ctx, backends)
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
		return b.Delegate.Elect(ctx, filtered)
	}

	if strategy == "any" {
		return b.Delegate.Elect(ctx, backends)
	}

	return nil, errors.New("Rejecting client due to not matching sni [" + sni + "].")

}
