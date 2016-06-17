/**
 * main.go - entry point
 * @author Yaroslav Pogrebnyak <yyyaroslav@gmail.com>
 */

package main

import (
	"./config"
	"./logging"
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
}

/**
 * Entry point
 */
func main() {

	flag.Parse()

	log := logging.For("main")
	log.Info("gobetween v", version)
	log.Info("Using config file ", configPath)

	var cfg config.Config
	if _, err := toml.DecodeFile(configPath, &cfg); err != nil {
		log.Fatal(err)
	}

	logging.Configure(cfg.Logging.Output, cfg.Logging.Level)

	// Begin work
	Start(cfg)
}
