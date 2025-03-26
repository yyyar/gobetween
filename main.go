/**
 * main.go - entry point
 * @author Yaroslav Pogrebnyak <yyyaroslav@gmail.com>
 */
package main

import (
	"log"
	"math/rand"
	"os"
	"runtime"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/yyyar/gobetween/api"
	"github.com/yyyar/gobetween/cmd"
	"github.com/yyyar/gobetween/config"
	"github.com/yyyar/gobetween/info"
	"github.com/yyyar/gobetween/logging"
	"github.com/yyyar/gobetween/manager"
	"github.com/yyyar/gobetween/metrics"
	"github.com/yyyar/gobetween/utils/codec"
)

/**
 * version,revision,branch should be set while build using ldflags (see Makefile)
 */
var (
	version  string
	revision string
	branch   string
)

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
	info.Revision = revision
	info.Branch = branch
	info.StartTime = time.Now()

}

/**
 * Entry point
 */
func main() {

	log.Printf("gobetween v%s", version)

	env := os.Getenv("GOBETWEEN")
	if env != "" && len(os.Args) > 1 {
		log.Fatal("Passed GOBETWEEN env var and command-line arguments: only one allowed")
	}

	// Try parse env var to args
	if env != "" {
		a := []string{}
		if err := codec.Decode(env, &a, "json"); err != nil {
			log.Fatal("Error converting env var to parameters: ", err, " ", env)
		}
		os.Args = append([]string{""}, a...)
		log.Println("Using parameters from env var: ", os.Args)
	}

	k8sShutdownTime := 0
	if v := os.Getenv("GOBETWEEN_SHUTDOWN_TIME"); v != "" {
		k8sShutdownTime, _ = strconv.Atoi(v)
		log.Printf("Using shutdown timeout: %s", time.Duration(k8sShutdownTime) * time.Second)
	}

	// Process flags and start
	cmd.Execute(func(cfg *config.Config) {

		// Configure logging
		logging.Configure(cfg.Logging.Output, cfg.Logging.Level, cfg.Logging.Format)

		// Start manager
		manager.Initialize(*cfg)

		/* setup metrics */
		metrics.Start((*cfg).Metrics)

		// Start API
		api.Start((*cfg).Api)
		
		// Wait to SIGTERM signal
		quit := make(chan os.Signal, 2)
		signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
		q := <-quit
		log.Printf("Received shutdown signal")
		if q == syscall.SIGTERM {
			log.Printf("Shutting down with timeout: %s", time.Duration(k8sShutdownTime) * time.Second)
			time.Sleep(time.Duration(k8sShutdownTime) * time.Second)
		}
	})
}
