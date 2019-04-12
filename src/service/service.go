package service

import (
	"github.com/yyyar/gobetween/config"
	"github.com/yyyar/gobetween/core"
	"github.com/yyyar/gobetween/logging"
)

/**
 * Registry of factory methods for Services
 */
var registry = make(map[string]func(config.Config) core.Service)

func All(cfg config.Config) []core.Service {
	log := logging.For("services")

	result := make([]core.Service, 0)

	for name, constructor := range registry {
		service := constructor(cfg)
		if service == nil {
			continue
		}
		log.Info("Creating ", name)
		result = append(result, service)
	}

	return result
}
