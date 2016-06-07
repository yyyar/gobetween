/**
 * exec.go - Exec healthcheck
 *
 * @author Yaroslav Pogrebnyak <yyyaroslav@gmail.com>
 */

package healthcheck

import (
	"../config"
	"../core"
	"../logging"
	"../utils"
	"time"
)

/**
 * Exec healthcheck
 */
func exec(t core.Target, cfg config.HealthcheckConfig, result chan<- CheckResult) {

	log := logging.For("healthcheck/exec")

	execTimeout, _ := time.ParseDuration(cfg.Timeout)

	checkResult := CheckResult{
		Target: t,
	}

	out, err := utils.ExecTimeout(execTimeout, cfg.ExecCommand, t.Host, t.Port)
	if err != nil {
		// TODO: Decide better what to do in this case
		checkResult.Live = false
		log.Warn(err)
	} else {
		if out == cfg.ExecExpectedPositiveOutput {
			checkResult.Live = true
		} else if out == cfg.ExecExpectedNegativeOutput {
			checkResult.Live = false
		} else {
			log.Warn("Unexpected output: ", out)
		}
	}

	select {
	case result <- checkResult:
	default:
		log.Warn("Channel is full. Discarding value")
	}
}
