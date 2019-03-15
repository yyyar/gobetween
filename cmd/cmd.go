package cmd

/**
 * cmd.go - command line runner
 *
 * @author Yaroslav Pogrebnyak <yyyaroslav@gmail.com>
 */

import (
	"github.com/yyyar/gobetween/config"
)

/**
 * App Start function to call after initialization
 */
var start func(*config.Config)

/**
 * Execute processing flags
 */
func Execute(f func(*config.Config)) {
	start = f
	RootCmd.Execute()
}
