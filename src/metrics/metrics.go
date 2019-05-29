package metrics

import (
	"fmt"
	"net/http"
	"runtime"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/yyyar/gobetween/config"
	"github.com/yyyar/gobetween/core"
	"github.com/yyyar/gobetween/info"
	"github.com/yyyar/gobetween/logging"
	"github.com/yyyar/gobetween/stats/counters"
)

const (
	namespace = "gobetween"
)

var (
	metricsDisabled bool = false
	log                  = logging.For("metrics")

	buildInfo *prometheus.GaugeVec
	version   string
	revision  string
	branch    string

	serverCount             *prometheus.GaugeVec
	serverActiveConnections *prometheus.GaugeVec
	serverRxTotal           *prometheus.GaugeVec
	serverTxTotal           *prometheus.GaugeVec
	serverRxSecond          *prometheus.GaugeVec
	serverTxSecond          *prometheus.GaugeVec

	backendActiveConnections  *prometheus.GaugeVec
	backendRefusedConnections *prometheus.GaugeVec
	backendTotalConnections   *prometheus.GaugeVec
	backendRxBytes            *prometheus.GaugeVec
	backendTxBytes            *prometheus.GaugeVec
	backendRxSecond           *prometheus.GaugeVec
	backendTxSecond           *prometheus.GaugeVec
	backendLive               *prometheus.GaugeVec
)

func defineMetrics() {

	buildInfo = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: namespace,
		Name:      "build_info",
		Help: fmt.Sprintf(
			"A metric with a constant '1' value labeled by version, revision, branch, and goversion from which %s was built.",
			namespace,
		),
	}, []string{"version", "revision", "branch", "goversion"})

	serverCount = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: namespace,
		Subsystem: "server",
		Name:      "count",
		Help:      "Server Count.",
	}, []string{"server"})

	serverActiveConnections = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: namespace,
		Subsystem: "server",
		Name:      "active_connections",
		Help:      "Server Actice Connections.",
	}, []string{"server"})

	serverRxTotal = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: namespace,
		Subsystem: "server",
		Name:      "rx_total",
		Help:      "Server Rx Total.",
	}, []string{"server"})

	serverTxTotal = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: namespace,
		Subsystem: "server",
		Name:      "tx_total",
		Help:      "Server Tx Total.",
	}, []string{"server"})

	serverRxSecond = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: namespace,
		Subsystem: "server",
		Name:      "rx_second",
		Help:      "Server Rx per Second.",
	}, []string{"server"})

	serverTxSecond = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: namespace,
		Subsystem: "server",
		Name:      "tx_second",
		Help:      "Server Tx per Second.",
	}, []string{"server"})

	backendActiveConnections = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: namespace,
		Subsystem: "backend",
		Name:      "active_connections",
		Help:      "Backend Actice Connections.",
	}, []string{"server", "host", "port"})

	backendRefusedConnections = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: namespace,
		Subsystem: "backend",
		Name:      "refused_connections",
		Help:      "Backend Refused Connections.",
	}, []string{"server", "host", "port"})

	backendTotalConnections = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: namespace,
		Subsystem: "backend",
		Name:      "total_connections",
		Help:      "Backend Total Connections.",
	}, []string{"server", "host", "port"})

	backendRxBytes = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: namespace,
		Subsystem: "backend",
		Name:      "rx_bytes",
		Help:      "Backend Rx Bytes.",
	}, []string{"server", "host", "port"})

	backendTxBytes = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: namespace,
		Subsystem: "backend",
		Name:      "tx_bytes",
		Help:      "Backend Tx Bytes.",
	}, []string{"server", "host", "port"})

	backendRxSecond = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: namespace,
		Subsystem: "backend",
		Name:      "rx_second",
		Help:      "Backend Rx per Second.",
	}, []string{"server", "host", "port"})

	backendTxSecond = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: namespace,
		Subsystem: "backend",
		Name:      "tx_second",
		Help:      "Backend Tx per Second.",
	}, []string{"server", "host", "port"})

	backendLive = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: namespace,
		Subsystem: "backend",
		Name:      "live",
		Help:      "Backend Alive.",
	}, []string{"server", "host", "port"})

}

func Start(cfg config.MetricsConfig) {

	if !cfg.Enabled {
		log.Info("Metrics disabled")
		metricsDisabled = true
		return
	}

	log.Info("Starting up Metrics server ", cfg.Bind)
	defineMetrics()

	prometheus.MustRegister(buildInfo)
	buildInfo.WithLabelValues(info.Version, info.Revision, info.Branch, runtime.Version()).Set(1)

	prometheus.MustRegister(serverCount)
	prometheus.MustRegister(serverActiveConnections)
	prometheus.MustRegister(serverRxTotal)
	prometheus.MustRegister(serverTxTotal)
	prometheus.MustRegister(serverRxSecond)
	prometheus.MustRegister(serverTxSecond)

	prometheus.MustRegister(backendActiveConnections)
	prometheus.MustRegister(backendRefusedConnections)
	prometheus.MustRegister(backendTotalConnections)
	prometheus.MustRegister(backendRxBytes)
	prometheus.MustRegister(backendTxBytes)
	prometheus.MustRegister(backendRxSecond)
	prometheus.MustRegister(backendTxSecond)
	prometheus.MustRegister(backendLive)

	http.Handle("/metrics", promhttp.Handler())
	go func() {
		fmt.Errorf("%s", http.ListenAndServe(cfg.Bind, nil))
	}()
}

func RemoveServer(server string, backends map[core.Target]*core.Backend) {
	if metricsDisabled {
		return
	}

	serverCount.DeleteLabelValues(server)
	serverActiveConnections.DeleteLabelValues(server)
	serverRxTotal.DeleteLabelValues(server)
	serverTxTotal.DeleteLabelValues(server)
	serverRxSecond.DeleteLabelValues(server)
	serverTxSecond.DeleteLabelValues(server)

	for _, backend := range backends {
		RemoveBackend(server, backend)
	}
}

func RemoveBackend(server string, backend *core.Backend) {
	if metricsDisabled {
		return
	}

	backendActiveConnections.DeleteLabelValues(server, backend.Host, backend.Port)
	backendRefusedConnections.DeleteLabelValues(server, backend.Host, backend.Port)
	backendTotalConnections.DeleteLabelValues(server, backend.Host, backend.Port)
	backendRxBytes.DeleteLabelValues(server, backend.Host, backend.Port)
	backendTxBytes.DeleteLabelValues(server, backend.Host, backend.Port)
	backendRxSecond.DeleteLabelValues(server, backend.Host, backend.Port)
	backendTxSecond.DeleteLabelValues(server, backend.Host, backend.Port)
	backendLive.DeleteLabelValues(server, backend.Host, backend.Port)
}

func ReportHandleBackendLiveChange(server string, target core.Target, live bool) {
	if metricsDisabled {
		return
	}

	intLive := int(0)
	if live {
		intLive = 1
	}

	backendLive.WithLabelValues(server, target.Host, target.Port).Set(float64(intLive))
}

func ReportHandleConnectionsChange(server string, connections uint) {
	if metricsDisabled {
		return
	}

	serverActiveConnections.WithLabelValues(server).Set(float64(connections))
}

func ReportHandleStatsChange(server string, bs counters.BandwidthStats) {
	if metricsDisabled {
		return
	}

	serverRxTotal.WithLabelValues(server).Set(float64(bs.RxTotal))
	serverTxTotal.WithLabelValues(server).Set(float64(bs.TxTotal))
	serverRxSecond.WithLabelValues(server).Set(float64(bs.RxSecond))
	serverTxSecond.WithLabelValues(server).Set(float64(bs.TxSecond))
}

func ReportHandleBackendStatsChange(server string, target core.Target, backends map[core.Target]*core.Backend) {
	if metricsDisabled {
		return
	}

	backend, _ := backends[target]

	serverCount.WithLabelValues(server).Set(float64(len(backends)))

	backendRxBytes.WithLabelValues(server, target.Host, target.Port).Set(float64(backend.Stats.RxBytes))
	backendTxBytes.WithLabelValues(server, target.Host, target.Port).Set(float64(backend.Stats.TxBytes))
	backendRxSecond.WithLabelValues(server, target.Host, target.Port).Set(float64(backend.Stats.RxSecond))
	backendTxSecond.WithLabelValues(server, target.Host, target.Port).Set(float64(backend.Stats.TxSecond))
}

func ReportHandleOp(server string, target core.Target, backends map[core.Target]*core.Backend) {
	if metricsDisabled {
		return
	}

	backend, _ := backends[target]

	backendActiveConnections.WithLabelValues(server, target.Host, target.Port).Set(float64(backend.Stats.ActiveConnections))
	backendRefusedConnections.WithLabelValues(server, target.Host, target.Port).Set(float64(backend.Stats.RefusedConnections))
	backendTotalConnections.WithLabelValues(server, target.Host, target.Port).Set(float64(backend.Stats.TotalConnections))
}
