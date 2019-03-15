package healthcheck

/**
 * ping.go - TCP ping healthcheck
 *
 * @author Yaroslav Pogrebnyak <yyyaroslav@gmail.com>
 */

import (
	"net"
	"time"

	"github.com/yyyar/gobetween/config"
	"github.com/yyyar/gobetween/core"
	"github.com/yyyar/gobetween/logging"
)

/**
 * Ping healthcheck
 */
func ping(t core.Target, cfg config.HealthcheckConfig, result chan<- CheckResult) {

	pingTimeoutDuration, _ := time.ParseDuration(cfg.Timeout)

	log := logging.For("healthcheck/ping")

	checkResult := CheckResult{
		Target: t,
	}

	conn, err := net.DialTimeout("tcp", t.Address(), pingTimeoutDuration)
	if err != nil {
		checkResult.Live = false
	} else {
		checkResult.Live = true
		conn.Close()
	}

	select {
	case result <- checkResult:
	default:
		log.Warn("Channel is full. Discarding value")
	}
}
