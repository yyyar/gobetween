package healthcheck

/**
 * worker.go - Healtheck worker
 *
 * @author Yaroslav Pogrebnyak <yyyaroslav@gmail.com>
 */

import (
	"time"

	"github.com/yyyar/gobetween/config"
	"github.com/yyyar/gobetween/core"
	"github.com/yyyar/gobetween/logging"
)

/**
 * Healthcheck Worker
 * Handles all periodic healthcheck logic
 * and yields results on change
 */
type Worker struct {

	/* Target to monitor and check */
	target core.Target

	/* Function that does actual check */
	check CheckFunc

	/* Channel to write changed check results */
	out chan<- CheckResult

	/* Healthcheck configuration */
	cfg config.HealthcheckConfig

	/* Stop channel to worker to stop */
	stop chan bool

	/* Last confirmed check result */
	LastResult CheckResult

	/* Current passes count, if LastResult.Live = true */
	passes int

	/* Current fails count, if LastResult.Live = false */
	fails int
}

/**
 * Start worker
 */
func (this *Worker) Start() {

	log := logging.For("healthcheck/worker")

	// Special case for no healthcheck, don't actually start worker
	if this.cfg.Kind == "none" {
		return
	}

	interval, _ := time.ParseDuration(this.cfg.Interval)

	ticker := time.NewTicker(interval)
	c := make(chan CheckResult, 1)

	go func() {
		for {
			select {

			/* new check interval has reached */
			case <-ticker.C:
				log.Debug("Next check ", this.cfg.Kind, " for ", this.target)
				go this.check(this.target, this.cfg, c)

			/* new check result is ready */
			case checkResult := <-c:
				log.Debug("Got check result ", this.cfg.Kind, ": ", checkResult)
				this.process(checkResult)

			/* request to stop worker */
			case <-this.stop:
				ticker.Stop()
				//close(c) // TODO: Check!
				return
			}
		}
	}()
}

/**
 * Process next check result,
 * counting passes and fails as needed, and
 * sending updated check result to out
 */
func (this *Worker) process(checkResult CheckResult) {

	log := logging.For("healthcheck/worker")

	if this.LastResult.Live && !checkResult.Live {
		this.passes = 0
		this.fails++
	} else if !this.LastResult.Live && checkResult.Live {
		this.fails = 0
		this.passes++
	} else {
		// check status not changed
		return
	}

	if this.passes == 0 && this.fails >= this.cfg.Fails ||
		this.fails == 0 && this.passes >= this.cfg.Passes {
		this.LastResult = checkResult

		log.Info("Sending to scheduler: ", this.LastResult)
		this.out <- checkResult
	}
}

/**
 * Stop worker
 */
func (this *Worker) Stop() {
	close(this.stop)
}
