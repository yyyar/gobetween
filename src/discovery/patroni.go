package discovery

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/sirupsen/logrus"
	"github.com/yyyar/gobetween/config"
	"github.com/yyyar/gobetween/core"
	"github.com/yyyar/gobetween/logging"
	"go.etcd.io/etcd/client"
	"net/url"
	"path"
	"strings"
	"sync"
	"time"
)

var patroni_manager *PatroniManager
var patroni_manager_mutex sync.Mutex

type PatroniManager struct {
	Lock sync.Mutex
	Clusters map[string]*PatroniCluster
}

type PatroniCluster struct {
	cfg       config.DiscoveryConfig
	Cluster   string
	Namespace string
	Members   map[string]*PatroniMember
	Leader    string

	Mutex 		sync.Mutex
	stop      context.CancelFunc
	ctxt      context.Context
	LeaderPools map[chan ([]core.Backend)]struct{}
	ReplicaPools map[chan ([]core.Backend)]struct{}
}

func GetPatroniManager() *PatroniManager {
	patroni_manager_mutex.Lock()
	defer patroni_manager_mutex.Unlock()

	if patroni_manager == nil {
		patroni_manager = NewPatroniManager()
	}
	return patroni_manager
}

func NewPatroniManager() *PatroniManager {
	return &PatroniManager{
		Clusters: make(map[string]*PatroniCluster),
	}
}

func (pm *PatroniManager) GetOrCreateCluster(cfg config.DiscoveryConfig) *PatroniCluster {
	/**
	 * This does not guarantee that equivalent definitions hit the same key, but using the same monitor is
	 * just a performance optimization
	 **/
	clusterKey := strings.Join(cfg.EtcdHosts, ",") + "-" + cfg.PatroniNamespace + cfg.PatroniCluster

	pm.Lock.Lock()
	defer pm.Lock.Unlock()

	cluster, ok := pm.Clusters[clusterKey]
	if !ok {
		cluster = NewPatroniCluster(cfg)
		pm.Clusters[clusterKey] = cluster
	}
	return cluster
}

func NewPatroniCluster(cfg config.DiscoveryConfig) *PatroniCluster {
	cl := PatroniCluster{
		cfg:       cfg,
		Cluster:   cfg.PatroniCluster,
		Namespace: cfg.PatroniNamespace,
		Members:   make(map[string]*PatroniMember),
		Leader:	   "",

		LeaderPools: make(map[chan ([]core.Backend)]struct{}),
		ReplicaPools: make(map[chan ([]core.Backend)]struct{}),
	}
	return &cl
}

func (cl *PatroniCluster) RunDiscovery(cfg config.DiscoveryConfig, out chan ([]core.Backend), stop chan bool) {
	cl.Mutex.Lock()
	/* Add ourselves to pools to be notified */
	switch cfg.PatroniPoolType {
	case "leader":
		cl.LeaderPools[out] = struct{}{}
	case "replica":
		cl.ReplicaPools[out] = struct{}{}
	}
	/* Start monitor if not yet running */
	if !cl.Running() {
		cl.Start()
	}
	cl.Mutex.Unlock()

	select {
	case <-stop:
		cl.Mutex.Lock()
		/* Remove from pool */
		switch cfg.PatroniPoolType {
		case "leader":
			delete(cl.LeaderPools, out)
		case "replica":
			delete(cl.ReplicaPools, out)
		}
		/* Stop monitor while no one is looking */
		if cl.Empty() {
			cl.Stop()
		}
		cl.Mutex.Unlock()
	}
}


/* Following method must be called while holding mutex */
func (cl *PatroniCluster) Start() {
	cl.ctxt, cl.stop = context.WithCancel(context.Background())

	go cl.MonitorEtcd()
}

func (cl *PatroniCluster) Stop() {
	cl.stop()
	cl.ctxt = nil
	cl.stop = nil
}

func (cl *PatroniCluster) Running() bool {
	return cl.ctxt != nil
}

func (cl *PatroniCluster) Empty() bool {
	return len(cl.LeaderPools) + len(cl.ReplicaPools) == 0
}


type PatroniMember struct {
	Name string
	ConnUrl string `json:"conn_url"`
	ApiUrl string `json:"api_url"`
	/* Don't care about these attributes for now
	State string `json:"state"`
	Role string `json:"role"`
	Version string `json:"version"`
	XlogLocation uint64 `json:"xlog_location"`
	Timeline string `json:"timeline"`
	 */

	target 		core.Target
}

func parsePatroniMember(node *client.Node) (*PatroniMember, error) {
	var member PatroniMember
	err := json.Unmarshal([]byte(node.Value), &member)
	if err != nil {
		return nil, err
	}
	member.parseConnUrl()
	return &member, nil
}

func (m *PatroniMember) Update(newMember *PatroniMember) (changed bool) {
	if m.ConnUrl != newMember.ConnUrl {
		m.ConnUrl = newMember.ConnUrl
		m.parseConnUrl()
		changed = true
	}
	if m.ApiUrl != newMember.ApiUrl {
		m.ApiUrl = newMember.ApiUrl
		changed = true
	}
	return
}

func (m *PatroniMember) Valid() bool {
	return m.target.Host != "" && m.target.Port != ""
}

func (m *PatroniMember) parseConnUrl() {
	dsn, err := url.Parse(m.ConnUrl)
	if err != nil {
		logrus.Warnf("Invalid connstring %v for %v", m.ConnUrl, m.Name)
	}
	m.target = core.Target{
		Host: dsn.Hostname(),
		Port: dsn.Port(),
	}
}



func (m *PatroniMember) Backend() core.Backend {
	return core.Backend{
		Target:   m.target,
		Priority: 1,
		Weight:   1,
		Stats:    core.BackendStats{Live: true},
	}
}

func (cl *PatroniCluster) MonitorEtcd() {
	log := logging.For("patroniDiscovery")
	log.Infof("Starting etcd monitor for Patroni cluster %v", cl.Cluster)
	defer log.Infof("Stopping etcd monitor for Patroni cluster %v", cl.Cluster)
	/* First iteration will not wait */
	timeout := 0*time.Second
	mainLoop:
	for {
		select {
		case <-cl.ctxt.Done():
			return
		case <-time.After(timeout):
			if timeout == 0 {
				timeout, _ = time.ParseDuration(cl.cfg.Timeout)
				if timeout == 0 {
					timeout = 10*time.Second
				}
			}
		}

		cli, err := createEtcdClient(cl.cfg)
		if err != nil {
			log.Warnf("Error creating etcd client: %v", err)
			continue
		}

		kapi := client.NewKeysAPI(cli)

		etcdIndex, err := cl.FetchMembers(kapi)
		if err != nil {
			log.Warnf("Error fetching cluster members: %v", err)
			continue
		}
		cl.FetchLeader(kapi)

		cl.UpdateListeners()

		watcher := kapi.Watcher(cl.MainPath(), &client.WatcherOptions{AfterIndex: etcdIndex, Recursive: true,})
		for {
			event, err := watcher.Next(cl.ctxt)
			if err != nil {
				log.Warnf("Error watching changes from etcd: %v", err)
				continue mainLoop
			}

			changed := false
			if strings.HasPrefix(event.Node.Key, cl.MemberPath()) {
				key := getKeyFromNode(event.Node)
				existingMember, ok := cl.Members[key]
				if event.Node.Value != "" {
					newMember, err := parsePatroniMember(event.Node)
					if err != nil {
						continue
					}
					if !ok {
						cl.Members[newMember.Name] = newMember
						changed = true
					} else {
						changed = existingMember.Update(newMember)
					}
				} else if (ok) {
					changed = true
					delete(cl.Members, key)
				}
			} else if strings.HasPrefix(event.Node.Key, cl.LeaderPath()) {
				if event.Node.Value != cl.Leader {
					cl.Leader = event.Node.Value
					changed = true
				}
			}
			if changed {
				cl.UpdateListeners()
			}
		}
	}
}

func (cl *PatroniCluster) FetchMembers(kapi client.KeysAPI) (uint64, error) {
	response, err := kapi.Get(cl.ctxt, cl.MemberPath(), &client.GetOptions{Recursive: true})
	if err != nil  {
		return 0, err
	}
	if !response.Node.Dir {
		return 0, errors.New(fmt.Sprintf("%v is not a directory", cl.MemberPath()))
	}
	for _, node := range response.Node.Nodes {
		key := getKeyFromNode(node)
		member, err := parsePatroniMember(node)
		if err != nil {
			logrus.Warnf("Invalid Patroni member %v: %v", key, err)
		}
		member.Name = key
		cl.Members[key] = member
	}
	return response.Index, nil
}

func (cl *PatroniCluster) UpdateListeners() {
	cl.Mutex.Lock()
	defer cl.Mutex.Unlock()
	if len(cl.LeaderPools) > 0 {
		backends := make([]core.Backend, 0, 1)
		if member, ok := cl.Members[cl.Leader]; ok && member.Valid() {
			backends = append(backends, member.Backend())
		}
		for pool, _ := range cl.LeaderPools {
			pool <- backends
		}
	}
	if len(cl.ReplicaPools) > 0 {
		backends := make([]core.Backend, 0, len(cl.Members))
		for _, member := range cl.Members {
			if member.Valid() {
				backend := member.Backend()
				if member.Name == cl.Leader {
					backend.Priority = 2
				}
				backends = append(backends, backend)
			}
		}
		for pool, _ := range cl.ReplicaPools {
			pool <- backends
		}
	}
}

func (cl *PatroniCluster) FetchLeader(kapi client.KeysAPI) {
	response, err := kapi.Get(cl.ctxt, cl.LeaderPath(), &client.GetOptions{Recursive: true})
	if err != nil {
		return
	}
	cl.Leader = response.Node.Value
}

func (cl *PatroniCluster) MainPath() string {
	return path.Join(cl.cfg.PatroniNamespace, cl.cfg.PatroniCluster)
}

func (cl *PatroniCluster) MemberPath() string {
	return path.Join(cl.MainPath(), "members")
}

func (cl *PatroniCluster) LeaderPath() string {
	return path.Join(cl.MainPath(), "leader")
}


func NewPatroniDiscovery(cfg config.DiscoveryConfig) interface{} {
	d := Discovery{
		opts:  DiscoveryOpts{1},
		watch: patroniWatch,
		cfg:   cfg,
	}

	return &d
}

func patroniWatch(cfg config.DiscoveryConfig, out chan ([]core.Backend), stop chan bool) {
	pm := GetPatroniManager()
	cl := pm.GetOrCreateCluster(cfg)
	cl.RunDiscovery(cfg, out, stop)
}