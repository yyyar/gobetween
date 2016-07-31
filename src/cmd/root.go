/**
 * root.go - root cmd emulates from-file TODO: remove when time will come
 *
 * @author Yaroslav Pogrebnyak <yyyaroslav@gmail.com>
 */
package cmd

import (
	"github.com/spf13/cobra"
)

/* Persistent parsed options */
var format string

/* Parsed options */
var configPath string

/**
 * Add Root Command
 */
func init() {
	RootCmd.Flags().StringVarP(&configPath, "config", "c", "", "Path to configuration file")
	RootCmd.PersistentFlags().StringVarP(&format, "format", "f", "toml", "Configuration file format: \"toml\" or \"json\"")
}

/**
 * Root Command
 */
var RootCmd = &cobra.Command{
	Use:   "gobetween",
	Short: "Modern & minimalistic load balancer for the Cload era",
	Run: func(cmd *cobra.Command, args []string) {

		if configPath == "" {
			cmd.Help()
			return
		}

		FromFileCmd.Run(cmd, []string{configPath})
	},
}
