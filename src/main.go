/**
 * main.go - entry point
 * @author Yaroslav Pogrebnyak <yyyaroslav@gmail.com>
 */
package main

import (
	"./api"
	"./cmd"
	"./config"
	"./info"
	"./logging"
	"./manager"
	"math/rand"
	"os"
	"runtime"
	"time"
)

/**
 * Version should be set while build using ldflags (see Makefile)
 */
var version string

/**
 * Initialize package
 */
func init() {

	// Set GOMAXPROCS if not set
	if os.Getenv("GOMAXPROCS") == "" {
		runtime.GOMAXPROCS(runtime.NumCPU())
	}

	// Init random seed
	rand.Seed(time.Now().UnixNano())

	// Save info
	info.Version = version
	info.StartTime = time.Now()

}

/**
 * Entry point
 */
func main() {

	// Process flags and start
	cmd.Execute(func(cfg *config.Config) {

		// Configure logging
		logging.Configure(cfg.Logging.Output, cfg.Logging.Level)

		// Start API
		go api.Start((*cfg).Api)

		// Start manager
		go manager.Initialize(*cfg)

		// block forever
		<-(chan string)(nil)
	})
}
