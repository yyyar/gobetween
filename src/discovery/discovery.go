package discovery

/**
 * discovery.go - discovery
 *
 * @author Yaroslav Pogrebnyak <yyyaroslav@gmail.com>
 */

import (
	"time"

	"github.com/yyyar/gobetween/config"
	"github.com/yyyar/gobetween/core"
	"github.com/yyyar/gobetween/logging"
)

/**
 * Registry of factory methods for Discoveries
 */
var registry = make(map[string]func(config.DiscoveryConfig) interface{})

/**
 * Initialize type registry
 */
func init() {
	registry["static"] = NewStaticDiscovery
	registry["srv"] = NewSrvDiscovery
	registry["docker"] = NewDockerDiscovery
	registry["json"] = NewJsonDiscovery
	registry["exec"] = NewExecDiscovery
	registry["plaintext"] = NewPlaintextDiscovery
	registry["consul"] = NewConsulDiscovery
	registry["lxd"] = NewLXDDiscovery
}

/**
 * Create new Discovery based on strategy
 */
func New(strategy string, cfg config.DiscoveryConfig) *Discovery {
	return registry[strategy](cfg).(*Discovery)
}

/**
 * Fetch func for pullig backends
 */
type FetchFunc func(config.DiscoveryConfig) (*[]core.Backend, error)

/**
 * Options for pull discovery
 */
type DiscoveryOpts struct {
	RetryWaitDuration time.Duration
}

/**
 * Discovery
 */
type Discovery struct {

	/**
	 * Cached backends
	 */
	backends *[]core.Backend

	/**
	 * Function to fetch / discovery backends
	 */
	fetch FetchFunc

	/**
	 * Options for fetch
	 */
	opts DiscoveryOpts

	/**
	 * Discovery configuration
	 */
	cfg config.DiscoveryConfig

	/**
	 * Channel where to push newly discovered backends
	 */
	out chan ([]core.Backend)

	/**
	 * Channel for stopping discovery
	 */
	stop chan bool
}

/**
 * Pull / fetch backends loop
 */
func (this *Discovery) Start() {

	log := logging.For("discovery")

	this.out = make(chan []core.Backend)
	this.stop = make(chan bool)

	// Prepare interval
	interval, err := time.ParseDuration(this.cfg.Interval)
	if err != nil {
		log.Fatal(err)
	}

	// TODO: rewrite with channels for stop
	go func() {
		for {
			backends, err := this.fetch(this.cfg)

			select {
			case <-this.stop:
				log.Info("Stopping discovery ", this.cfg)
				return
			default:
			}

			if err != nil {
				log.Error(this.cfg.Kind, " error ", err, " retrying in ", this.opts.RetryWaitDuration.String())
				log.Info("Applying failpolicy ", this.cfg.Failpolicy)

				if this.cfg.Failpolicy == "setempty" {
					this.backends = &[]core.Backend{}
					if !this.send() {
						log.Info("Stopping discovery ", this.cfg)
						return
					}
				}

				if !this.wait(this.opts.RetryWaitDuration) {
					log.Info("Stopping discovery ", this.cfg)
					return
				}

				continue
			}

			// cache
			this.backends = backends
			if !this.send() {
				log.Info("Stopping discovery ", this.cfg)
				return
			}

			// exit gorouting if no cacheTtl
			// used for static discovery
			if interval == 0 {
				return
			}

			if !this.wait(interval) {
				log.Info("Stopping discovery ", this.cfg)
				return
			}
		}
	}()
}

func (this *Discovery) send() bool {
	// out if not stopped
	select {
	case <-this.stop:
		return false
	default:
		this.out <- *this.backends
		return true
	}
}

/**
 * wait waits for interval or stop
 * returns true if waiting was successfull
 * return false if waiting was interrupted with stop
 */
func (this *Discovery) wait(interval time.Duration) bool {

	t := time.NewTimer(interval)

	select {
	case <-t.C:
		return true

	case <-this.stop:
		if !t.Stop() {
			<-t.C
		}
		return false
	}

}

/**
 * Stop discovery
 */
func (this *Discovery) Stop() {
	this.stop <- true
}

/**
 * Returns backends channel
 */
func (this *Discovery) Discover() <-chan []core.Backend {
	return this.out
}
