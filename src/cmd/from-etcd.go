package cmd

/**
 * from-etcd.go - pull config from etcd and run
 *
 * @author Pavlo Golub <pavlo.golub@gmail.com>
 */

import (
	"context"
	"log"

	etcd "github.com/etcd-io/etcd/client"
	"github.com/spf13/cobra"
	"github.com/yyyar/gobetween/config"
	"github.com/yyyar/gobetween/info"
	"github.com/yyyar/gobetween/utils"
	"github.com/yyyar/gobetween/utils/codec"
)

/* Parsed options */
var etcdKey string
var etcdConfig etcd.Config = etcd.Config{}

/**
 * Add command
 */
func init() {

	FromEtcdCmd.Flags().StringVarP(&etcdKey, "key", "k", "gobetween", "Etcd Key to pull config from")

	RootCmd.AddCommand(FromEtcdCmd)
}

/**
 * FromConsul command
 */
var FromEtcdCmd = &cobra.Command{
	Use:   "from-etcd <host:port>",
	Short: "Start using config from etcd",
	Long:  `Start using config from the etcd key-value storage`,
	Run: func(cmd *cobra.Command, args []string) {

		if len(args) != 1 {
			_ = cmd.Help()
			return
		}

		etcdConfig.Endpoints = []string{args[0]}
		client, err := etcd.New(etcdConfig)
		if err != nil {
			log.Fatal(err)
		}
		kapi := etcd.NewKeysAPI(client)

		response, err := kapi.Get(context.Background(), etcdKey, &etcd.GetOptions{Recursive: true})
		if err != nil {
			log.Fatal("Error retrieving backends from etcd: ", err)
		}

		datastr := string(response.Node.Value)
		if isConfigEnvVars {
			datastr = utils.SubstituteEnvVars(datastr)
		}

		var cfg config.Config
		if err := codec.Decode(datastr, &cfg, format); err != nil {
			log.Fatal(err)
		}

		info.Configuration = struct {
			Kind string `json:"kind"`
			Host string `json:"host"`
			Key  string `json:"key"`
		}{"etcd", args[0], etcdKey}

		start(&cfg)
	},
}
