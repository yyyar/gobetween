package discovery

/**
 * etcd.go - Etcd API discovery implementation
 *
 * @author Ants Aasma <ants.aasma@eesti.ee>
 */

import (
	"context"
	"net/url"
	"strings"
	"time"

	"github.com/elgs/gojq"
	"github.com/etcd-io/etcd/pkg/transport"
	"github.com/sirupsen/logrus"

	"github.com/etcd-io/etcd/client"
	"github.com/yyyar/gobetween/config"
	"github.com/yyyar/gobetween/core"
	"github.com/yyyar/gobetween/logging"
)

type Host struct {
	address string
	port    string
}

/**
 * Create new Discovery with Json fetch func
 */
func NewEtcdDiscovery(cfg config.DiscoveryConfig) interface{} {
	d := Discovery{
		opts:  DiscoveryOpts{1},
		watch: etcdWatch,
		cfg:   cfg,
	}

	return &d
}

func etcdWatch(cfg config.DiscoveryConfig, out chan ([]core.Backend), stop chan bool) {
	log := logging.For("etcdWatch")
	retryTimeout := time.Second * 30

	ctxt, cancel := context.WithCancel(context.Background())
	go func() {
		select {
		case <-stop:
			cancel()
		}
	}()

mainLoop:
	for {
		cli, err := createEtcdClient(cfg)
		if err != nil {
			// handle error!
			log.Error("error connecting to etcd: ", err)
			if !waitForRetry(ctxt, retryTimeout) {
				return
			}
			continue
		}

		var members = make(map[string]Host)

		kapi := client.NewKeysAPI(cli)

		response, err := kapi.Get(ctxt, cfg.EtcdPrefix, &client.GetOptions{Recursive: true})
		if err != nil {
			log.Error("Error retrieving backends from etcd: ", err)
			if !waitForRetry(ctxt, retryTimeout) {
				return
			}
			continue
		}
		if !response.Node.Dir {
			log.Errorf("Prefix path %s is not a directory", cfg.EtcdPrefix)
		}
		for _, node := range response.Node.Nodes {
			key := getKeyFromNode(node)
			host, err := nodeToHost(cfg, node)
			if err != nil {
				log.Warn("Invalid node at %s: %s", node.Key, err)
				continue
			}
			members[key] = host
		}

		select {
		case <-ctxt.Done():
			return
		case out <- constructCluster(members, log):
		}

		watcher := kapi.Watcher(cfg.EtcdPrefix, &client.WatcherOptions{AfterIndex: response.Index, Recursive: true})
		for {
			event, err := watcher.Next(ctxt)
			if err != nil {
				log.Error("error watching cluster: ", err)
				continue mainLoop
			}
			updated := false

			key := getKeyFromNode(event.Node)
			if event.Action == "set" || event.Action == "create" {
				host, err := nodeToHost(cfg, event.Node)
				if err != nil {
					log.Warn("Invalid node at %s: %s", event.Node.Key, err)
					continue
				}
				if existing, ok := members[key]; ok {
					if existing.address != host.address || existing.port != host.port {
						updated = true
						log.Debug("Member ", key, " updated to ", host)
						members[key] = host
					}
				} else {
					updated = true
					members[key] = host
				}
			} else if event.Action == "delete" || event.Action == "expire" {
				if _, ok := members[key]; ok {
					log.Debug("Deleted member ", key)
					delete(members, key)
					updated = true
				}
			}
			if updated {
				select {
				case <-ctxt.Done():
					return
				case out <- constructCluster(members, log):
				}
			}
		}
	}
}

func getKeyFromNode(node *client.Node) string {
	path := strings.Split(node.Key, "/")
	return path[len(path)-1]
}

func createEtcdClient(cfg config.DiscoveryConfig) (client.Client, error) {
	timeout, _ := time.ParseDuration(cfg.Timeout)
	config := client.Config{
		Endpoints:               cfg.EtcdHosts,
		HeaderTimeoutPerRequest: timeout,
	}
	if cfg.EtcdUsername != nil {
		config.Username = *cfg.EtcdUsername
		config.Password = *cfg.EtcdPassword
	}

	if cfg.EtcdTlsEnabled {
		tls := transport.TLSInfo{
			CertFile:      cfg.EtcdTlsCertPath,
			KeyFile:       cfg.EtcdTlsKeyPath,
			TrustedCAFile: cfg.EtcdTlsCacertPath,
		}
		t, err := transport.NewTransport(tls, timeout)
		if err != nil {
			return nil, err
		}
		config.Transport = t
	} else {
		config.Transport = client.DefaultTransport
	}

	cli, err := client.New(config)
	return cli, err
}

func nodeToHost(cfg config.DiscoveryConfig, node *client.Node) (host Host, errout error) {
	parsed, err := gojq.NewStringQuery(string(node.Value))
	if err != nil {
		errout = err
		return
	}
	dsn, err := parsed.QueryToString(cfg.EtcdDsnJsonPath)
	if err != nil {
		errout = err
		return
	}
	dsnUrl, err := url.Parse(dsn)
	if err != nil {
		errout = err
		return
	}
	host = Host{address: dsnUrl.Hostname(), port: dsnUrl.Port()}
	return
}

func constructCluster(members map[string]Host, log *logrus.Entry) []core.Backend {
	backends := make([]core.Backend, 0, 0)
	for _, member := range members {
		backends = append(backends, core.Backend{
			Target: core.Target{
				Host: member.address,
				Port: member.port,
			},
			Priority: 1,
			Weight:   1,
			Stats: core.BackendStats{
				Live: true,
			},
		})
	}
	log.Debugf("Sending backend list: %s", backends)
	return backends
}

func waitForRetry(ctxt context.Context, retryTimeout time.Duration) bool {
	select {
	case <-ctxt.Done():
		return false
	case <-time.After(retryTimeout):
	}
	return true
}
