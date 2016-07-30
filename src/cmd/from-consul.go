/**
 * from-consul.go - pull config from consul and run
 *
 * @author Yaroslav Pogrebnyak <yyyaroslav@gmail.com>
 */
package cmd

import (
	"../config"
	"../utils/codec"
	consul "github.com/hashicorp/consul/api"
	"github.com/spf13/cobra"
	"log"
)

/* Parsed options */
var consulHost string
var consulKey string

/**
 * Add command
 */
func init() {

	FromConsulCmd.Flags().StringVarP(&consulHost, "host", "", "localhost", "Consul host")
	FromConsulCmd.Flags().StringVarP(&consulKey, "key", "", "gobetween", "Consul Key to pull config from")

	RootCmd.AddCommand(FromConsulCmd)
}

/**
 * FromConsul command
 */
var FromConsulCmd = &cobra.Command{
	Use:   "from-consul",
	Short: "Pull config from Consul",
	Long:  `Pull config from the Consul Key-Value storage`,
	Run: func(cmd *cobra.Command, args []string) {

		client, err := consul.NewClient(&consul.Config{
			Address: consulHost,
		})
		if err != nil {
			log.Fatal(err)
		}

		pair, _, err := client.KV().Get(consulKey, nil)
		if err != nil {
			log.Fatal(err)
		}

		if pair == nil {
			log.Fatal("Empty value for key " + consulKey)
		}

		var cfg config.Config
		if err := codec.Decode(string(pair.Value), &cfg, format); err != nil {
			log.Fatal(err)
		}

		start(&cfg)
	},
}
