/**
 * from-file.go - pull config from file and run
 *
 * @author Yaroslav Pogrebnyak <yyyaroslav@gmail.com>
 */
package cmd

import (
	"../config"
	"../utils/codec"
	"github.com/spf13/cobra"
	"io/ioutil"
	"log"
)

/* Parsed options */
var configPath string

/**
 * Add Root Command
 */
func init() {
	FromFileCmd.Flags().StringVarP(&configPath, "config", "c", "./gobetween.toml", "Path to configuration file")
	RootCmd.AddCommand(FromFileCmd)
}

/**
 * FromFile Command
 */
var FromFileCmd = &cobra.Command{
	Use:   "from-file",
	Short: "Use config from file",
	Run: func(cmd *cobra.Command, args []string) {

		data, err := ioutil.ReadFile(configPath)
		if err != nil {
			log.Fatal(err)
		}

		var cfg config.Config
		if err = codec.Decode(string(data), &cfg, format); err != nil {
			log.Fatal(err)
		}

		start(&cfg)
	},
}
