package discovery

import (
	"context"
	"github.com/prometheus/common/log"
	"time"
	"strings"
	"sort"
	"encoding/json"
	"net/url"
	"sync"
	"sync/atomic"

	"go.etcd.io/etcd/client"
	"github.com/yyyar/gobetween/config"
	"github.com/yyyar/gobetween/core"
	"github.com/yyyar/gobetween/logging"
)

type Host struct {
	address string
	port string
	isleader bool
}
type EtcdCluster struct {
	Leader   *string
	Members  map[string]Host
	Config   config.DiscoveryConfig
	Mutex    sync.Mutex
}
type EtcdWatcher struct {
	Clusters map[string]*EtcdCluster
}

var etcd_watcher_initialized uint32
var etcd_watcher_instance *EtcdWatcher
var etcd_instance_mutex sync.Mutex

func GetInstance() *EtcdWatcher {
	if atomic.LoadUint32(&etcd_watcher_initialized) == 1 {
		return etcd_watcher_instance
	}

	etcd_instance_mutex.Lock()
	defer etcd_instance_mutex.Unlock()

	if etcd_watcher_instance == nil {
		etcd_watcher_instance = &EtcdWatcher{Clusters: make(map[string]*EtcdCluster)}
		atomic.StoreUint32(&etcd_watcher_initialized, 1)
	}

	return etcd_watcher_instance
}

func (e *EtcdWatcher) AddPrefix(cfg config.DiscoveryConfig) {
	etcd_instance_mutex.Lock()
	defer etcd_instance_mutex.Unlock()

	logging.For("EtcdWatcher AddPrefix")
	_, present := e.Clusters[cfg.EtcdPrefix]

	if ! present {
		cluster := EtcdCluster{Config: cfg, Members: make(map[string]Host)}
		e.Clusters[cfg.EtcdPrefix] = &cluster
		go cluster.StartWatcher()
	}
}

func (e *EtcdWatcher) GetCluster(cfg config.DiscoveryConfig) *EtcdCluster {
	etcd_instance_mutex.Lock()
	defer etcd_instance_mutex.Unlock()

	if cluster, ok := e.Clusters[cfg.EtcdPrefix]; ok {
		return cluster
	}
	return nil
}

type PatroniMember struct {
	ConnUrl string `json:"conn_url"`
	ApiUrl string `json:"api_url"`
	State string `json:"state"`
	Role string `json:"role"`
	Version string `json:"version"`
	XlogLocation uint64 `json:"xlog_location"`
	Timeline string `json:"timeline"`
}

func nodeToHost(node *client.Node) (string, Host) {
	path := strings.Split(node.Key, "/")
	var member PatroniMember
	json.Unmarshal([]byte(node.Value), &member)
	log.Info("Member: ", member)
	url, _ := url.Parse(member.ConnUrl)

	return path[len(path) -1], Host{address: url.Hostname(), port: string(url.Port()),}
}

func (c *EtcdCluster) StartWatcher() {
	log := logging.For("EtcdWatcher WatchPrefix")
	config := client.Config{
		Endpoints: c.Config.EtcdHosts,
		Transport: client.DefaultTransport,
		HeaderTimeoutPerRequest: 5 * time.Second,
	}
	if c.Config.EtcdUsername != nil {
		config.Username = *c.Config.EtcdUsername
		config.Password = *c.Config.EtcdPassword
	}

	cli, err := client.New(config)
	if err != nil {
		// handle error!
		log.Error("error connecting to etcd: ", err)
		return;
	}

	var members = make(map[string]Host)

	kapi := client.NewKeysAPI(cli)
	response, err := kapi.Get(context.TODO(), c.Config.EtcdPrefix + "/members", &client.GetOptions{Recursive: true})
	if err != nil {
		log.Error("Error retrieving backends from etcd: ", err)
		return
	}
	if !response.Node.Dir {
		log.Error("Not a directory")
	}
	for _, node := range response.Node.Nodes {
		key, host := nodeToHost(node)
		log.Info("Got ", key, " = ", host)
		members[key] = host
	}
	c.SetMembers(members)

	response, err = kapi.Get(context.TODO(), c.Config.EtcdPrefix + "/leader", nil)
	if err == nil {
		log.Info("Leader", response.Node.Value)
		c.SetLeader(response.Node.Value)
	}

	watcher := kapi.Watcher(c.Config.EtcdPrefix, &client.WatcherOptions{AfterIndex: 0, Recursive: true,})
	for {
		event, err := watcher.Next(context.Background())
		if err != nil {
			log.Error("error watching cluster: ", err)
			continue // TODO: need to restart the watch here
		}

		if event.Node.Key == c.Config.EtcdPrefix + "/leader" {
			c.SetLeader(event.Node.Value)
			log.Info("leader: ", event.Node.Value)
			for k, v := range members {
				log.Info("member ", k, ": ", v)
			}
		} else if strings.HasPrefix(event.Node.Key, c.Config.EtcdPrefix + "/members/") {
			path, host := nodeToHost(event.Node)
			members[path] = host
			c.SetMembers(members)
		}
	}
}

func (c *EtcdCluster) GetLeader() *Host {
	c.Mutex.Lock()
	defer c.Mutex.Unlock()

	if c.Leader != nil {
		if member, ok := c.Members[*c.Leader]; ok {
			return &member
		}
	}
	return nil
}

func (c *EtcdCluster) GetMembers() map[string]Host {
	c.Mutex.Lock()
	defer c.Mutex.Unlock()

	return c.Members
}

func (c *EtcdCluster) SetLeader(name string) {
	c.Mutex.Lock()
	defer c.Mutex.Unlock()
	c.Leader = &name
}

func (c *EtcdCluster) SetMembers(hosts map[string]Host) {
	c.Mutex.Lock()
	defer c.Mutex.Unlock()
	c.Members = hosts
}

/**
 * Create new Discovery with Json fetch func
 */
func NewEtcdDiscovery(cfg config.DiscoveryConfig) interface{} {
	var watcher *EtcdWatcher
	watcher = GetInstance()
	watcher.AddPrefix(cfg)

	d := Discovery{
		opts:  DiscoveryOpts{1},
		//fetch: etcdFetch,
		watch: etcdWatch,
		cfg:   cfg,
	}

	return &d
}

func etcdWatch(cfg config.DiscoveryConfig, out chan ([]core.Backend), stop chan bool)  {
	log := logging.For("etcdWatch")
	retryTimeout := time.Second*30

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
		var leader *string = nil

		kapi := client.NewKeysAPI(cli)

		response, err := kapi.Get(ctxt, cfg.EtcdPrefix+"/members", &client.GetOptions{Recursive: true})
		if err != nil {
			log.Error("Error retrieving backends from etcd: ", err)
			return
		}
		if !response.Node.Dir {
			log.Error("Not a directory")
		}
		for _, node := range response.Node.Nodes {
			key, host := nodeToHost(node)
			log.Info("Got ", key, " = ", host)
			members[key] = host
		}
		response, err = kapi.Get(ctxt, cfg.EtcdPrefix+"/leader", nil)
		if err == nil {
			log.Info("Leader: ", response.Node.Value)
			leader = &response.Node.Value
		}

		select {
			case <-ctxt.Done():
				return
			case out <- constructCluster(cfg.EtcdLeaderPool, members, leader):
		}


		watcher := kapi.Watcher(cfg.EtcdPrefix, &client.WatcherOptions{AfterIndex: response.Index, Recursive: true,})
		for {
			event, err := watcher.Next(ctxt)
			if err != nil {
				log.Error("error watching cluster: ", err)
				continue mainLoop
			}
			updated := false

			log.Info("Got event ", event.Action, " on ", event.Node.Key)

			if event.Node.Key == cfg.EtcdPrefix+"/leader" {
				if event.Action == "delete" || event.Action == "compareAndDelete" || event.Action == "expire" {
					updated = leader != nil
					leader = nil
					if (updated) {
						log.Info("lost leader")
					}
				} else if event.Action == "set" || event.Action == "create" || event.Action == "compareAndSwap" {
					updated = leader == nil || *leader != event.Node.Value
					leader = &event.Node.Value
					if (updated) {
						log.Info("New leader ", *leader)
					}
				}
			} else if strings.HasPrefix(event.Node.Key, cfg.EtcdPrefix+"/members/") {
				path, host := nodeToHost(event.Node)
				if event.Action == "set" || event.Action == "create" {
					if existing, ok := members[path]; ok {
						if existing.address != host.address || existing.port != host.port {
							updated = true
							log.Info("Member ", path, " updated to ", host)
							members[path] = host
						}
					} else {
						updated = true
						members[path] = host
					}
				} else if event.Action == "delete" || event.Action == "expire"  {
					if _, ok := members[path]; ok {
						log.Info("Deleted member ", path)
						delete(members, path)
						updated = true
					}
				}
			}
			if (updated) {
				select {
				case <-ctxt.Done():
					return
				case out <- constructCluster(cfg.EtcdLeaderPool, members, leader):
				}
			}
		}
	}
}

func constructCluster(leaderPool bool, members map[string]Host, leader *string) []core.Backend {
	backends := make([]core.Backend, 0, 0)
	if leaderPool {
		if leader != nil {
			if member, ok := members[*leader]; ok {
				backends = append(backends, core.Backend{
					Target: core.Target{
						Host: member.address,
						Port: member.port,
					},
					Priority: 1,
					Weight: 1,
					Stats: core.BackendStats{
						Live: true,
					},
				})
			}
		}
	} else {
		includeLeader := false
		if leader != nil && len(members) == 1 {
			if _, ok := members[*leader]; ok {
				includeLeader = true
			}
		}

		for key, member := range members {
			if leader != nil && key == *leader && !includeLeader {
				continue
			}

			backends = append(backends, core.Backend{
				Target: core.Target{
					Host: member.address,
					Port: member.port,
				},
				Priority: 1,
				Weight: 1,
				Stats: core.BackendStats{
					Live: true,
				},
			})
		}
	}
	log.Info("", backends)
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

func createEtcdClient(cfg config.DiscoveryConfig) (client.Client, error) {
	config := client.Config{
		Endpoints:               cfg.EtcdHosts,
		Transport:               client.DefaultTransport,
		HeaderTimeoutPerRequest: 5 * time.Second,
	}
	if cfg.EtcdUsername != nil {
		config.Username = *cfg.EtcdUsername
		config.Password = *cfg.EtcdPassword
	}

	cli, err := client.New(config)
	return cli, err
}


/**
 * Fetch / refresh backends from URL with json in response
 */
func etcdFetch(cfg config.DiscoveryConfig) (*[]core.Backend, error) {

	log := logging.For("etcdFetch")
	var backends []core.Backend

	watcher := GetInstance()
	cluster := watcher.GetCluster(cfg)
	if cluster != nil {
		leader := cluster.GetLeader()

		if cfg.EtcdLeaderPool {
			/* leader */
			if leader != nil {
				backends = append(backends, core.Backend{
					Target: core.Target{
						Host: leader.address,
						Port: leader.port,
					},
					Priority: 1,
					Weight: 1,
					Stats: core.BackendStats{
						Live: true,
					},
					Sni: leader.address,
				})
			}
		} else {
			/* followers */
			for id, host := range cluster.GetMembers() {
				weight := cfg.EtcdFollowerWeight
				if leader != nil && host == *leader {
					weight = cfg.EtcdLeaderWeight
				}

				backends = append(backends, core.Backend{
					Target: core.Target{
						Host: host.address,
						Port: host.port,
					},
					Priority: 1,
					Weight: weight,
					Stats: core.BackendStats{
						Live: true,
					},
					Sni: id,
				})
			}
			// sort backends by host address
			sort.Slice(backends, func(i, j int) bool {
				return backends[i].Target.Host < backends[j].Target.Host
			})
		}
	} else {
		log.Error("No cluster with prefix ", cfg.EtcdPrefix)
	}

	log.Info("backends: ", backends)
	return &backends, nil
}
