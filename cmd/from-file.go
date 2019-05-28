package cmd

/**
 * from-file.go - pull config from file and run
 *
 * @author Yaroslav Pogrebnyak <yyyaroslav@gmail.com>
 */

import (
	"io/ioutil"
	"log"
	"os"
	"regexp"
	"strings"

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

		datastr := mapEnvVars(string(data))

		if err = codec.Decode(datastr, &cfg, format); err != nil {
			log.Fatal(err)
		}

		info.Configuration = struct {
			Kind string `json:"kind"`
			Path string `json:"path"`
		}{"file", args[0]}

		start(&cfg)
	},
}

//
// mapEnvVars replaces placeholders ${...} with env var value
//
func mapEnvVars(data string) string {

	var re = regexp.MustCompile(`\${.*?}`)

	vars := re.FindAllString(data, -1)
	for _, v := range vars {
		data = strings.ReplaceAll(data, v, os.Getenv(v[2:len(v)-1]))
	}
	return data
}
