package cmd

/**
 * from-file.go - pull config from file and run
 *
 * @author Yaroslav Pogrebnyak <yyyaroslav@gmail.com>
 */

import (
	"io/ioutil"
	"log"

	"github.com/spf13/cobra"
	"github.com/yyyar/gobetween/config"
	"github.com/yyyar/gobetween/info"
	"github.com/yyyar/gobetween/utils/codec"
)

/**
 * Add Root Command
 */
func init() {
	RootCmd.AddCommand(FromFileCmd)
}

/**
 * FromFile Command
 */
var FromFileCmd = &cobra.Command{
	Use:   "from-file <path>",
	Short: "Start using config from file",
	Run: func(cmd *cobra.Command, args []string) {

		if len(args) != 1 {
			cmd.Help()
			return
		}

		data, err := ioutil.ReadFile(args[0])
		if err != nil {
			log.Fatal(err)
		}

		var cfg config.Config
		if err = codec.Decode(string(data), &cfg, format); err != nil {
			log.Fatal(err)
		}

		info.Configuration = struct {
			Kind string `json:"kind"`
			Path string `json:"path"`
		}{"file", args[0]}

		start(&cfg)
	},
}
