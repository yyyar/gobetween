package profiler

import (
	"../../logging"
	"net/http"
	_ "net/http/pprof"
)

func Start(bind string) {
	log := logging.For("profiler")

	log.Infof("Starting profiler: %v", bind)

	go func() {
		log.Error(http.ListenAndServe(bind, nil))
	}()
}
