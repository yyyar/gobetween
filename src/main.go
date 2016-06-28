/**
 * main.go - entry point
 * @author Yaroslav Pogrebnyak <yyyaroslav@gmail.com>
 */

package main

import (
	"./api"
	"./config"
	"./info"
	"./logging"
	"./manager"
	"flag"
	"github.com/BurntSushi/toml"
	"math/rand"
	"os"
	"runtime"
	"time"
)

/**
 * Constants
 */
const (
	defaultConfigPath = "./gobetween.toml"
)

/**
 * Version should be set while build
 * using ldflags (see Makefile)
 */
var version string

var configPath string

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

	// Init command-line flags
	flag.StringVar(&configPath, "c", defaultConfigPath, "Path to config file")

	// Set info to be used in another parts of the program
	info.Version = version
	info.ConfigPath = configPath
	info.StartTime = time.Now()
}

/**
 * Entry point
 */
func main() {

	flag.Parse()

	log := logging.For("main")
	log.Info("gobetween v", version)
	log.Info("Using config file ", configPath)

	// Parse config
	var cfg config.Config
	if _, err := toml.DecodeFile(configPath, &cfg); err != nil {
		log.Fatal(err)
	}

	// Configure logging
	logging.Configure(cfg.Logging.Output, cfg.Logging.Level)

	// Start API
	go api.Start(cfg.Api)

	// Start manager
	go manager.Initialize(cfg)

	// block forever
	<-(chan string)(nil)
}
