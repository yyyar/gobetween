package cmd

/**
 * from-url.go - pull config from url and run
 *
 * @author Yaroslav Pogrebnyak <yyyaroslav@gmail.com>
 */

import (
	"io/ioutil"
	"log"
	"net/http"

	"github.com/spf13/cobra"
	"github.com/yyyar/gobetween/config"
	"github.com/yyyar/gobetween/info"
	"github.com/yyyar/gobetween/utils/codec"
)

/**
 * Add command
 */
func init() {

	RootCmd.AddCommand(FromUrlCmd)
}

/**
 * FromUrlCmd command
 */
var FromUrlCmd = &cobra.Command{
	Use:   "from-url <url>",
	Short: "Start using config from URL",
	Run: func(cmd *cobra.Command, args []string) {

		if len(args) != 1 {
			cmd.Help()
			return
		}

		client := http.Client{}
		res, err := client.Get(args[0])
		if err != nil {
			log.Fatal(err)
		}

		defer res.Body.Close()

		// Read response
		content, err := ioutil.ReadAll(res.Body)
		if err != nil {
			log.Fatal(err)
		}

		var cfg config.Config
		if err := codec.Decode(string(content), &cfg, format); err != nil {
			log.Fatal(err)
		}

		info.Configuration = struct {
			Kind string `json:"kind"`
			Url  string `json:"url"`
		}{"url", args[0]}

		start(&cfg)
	},
}
