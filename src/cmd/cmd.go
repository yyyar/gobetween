/**
 * cmd.go - command line runner
 *
 * @author Yaroslav Pogrebnyak <yyyaroslav@gmail.com>
 */
package cmd

import (
	"../config"
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
