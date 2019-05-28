package cmd

/**
 * root.go - root cmd emulates from-file TODO: remove when time will come
 *
 * @author Yaroslav Pogrebnyak <yyyaroslav@gmail.com>
 */

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/yyyar/gobetween/info"
)

/* Persistent parsed options */
var format string

/* Parsed options */
var configPath string

/* Show version */
var showVersion bool

/* Substitute env vars in config or not */
var isConfigEnvVars bool

/**
 * Add Root Command
 */
func init() {
	RootCmd.Flags().BoolVarP(&showVersion, "version", "v", false, "Print version information and quit")
	RootCmd.Flags().StringVarP(&configPath, "config", "c", "", "Path to configuration file")
	RootCmd.PersistentFlags().StringVarP(&format, "format", "f", "toml", "Configuration file format: \"toml\" or \"json\"")
	RootCmd.PersistentFlags().BoolVarP(&isConfigEnvVars, "use-config-env-vars", "e", false, "Enable env variables interpretation in config file")
}

/**
 * Root Command
 */
var RootCmd = &cobra.Command{
	Use:   "gobetween",
	Short: "Modern & minimalistic load balancer for the Cloud era",
	Run: func(cmd *cobra.Command, args []string) {

		if showVersion {
			fmt.Println(info.Version)
			return
		}

		if configPath == "" {
			cmd.Help()
			return
		}

		FromFileCmd.Run(cmd, []string{configPath})
	},
}
