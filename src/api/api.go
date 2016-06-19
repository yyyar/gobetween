/**
 * api.go - rest api implementation
 *
 * @author Yaroslav Pogrebnyak <yyyaroslav@gmail.com>
 */
package api

import (
	"../config"
	"../logging"
	"github.com/go-zoo/bone"
	"io"
	"net/http"
	"os"
	"time"
)

/**
 * Time when server was started.
 * TODO: Probably move to better place.
 */
var startTime time.Time = time.Now()

/**
 * Starts REST API server
 */
func Start(cfg config.ApiConfig, servers interface{}) {

	var log = logging.For("api")

	if !cfg.Enabled {
		log.Info("API disabled")
		return
	}

	log.Info("Starting up API")

	mux := bone.New()

	mux.GetFunc("/", func(w http.ResponseWriter, req *http.Request) {

		io.WriteString(w, Marshal(map[string]interface{}{
			"pid":       os.Getpid(),
			"time":      time.Now(),
			"startedAt": startTime,
			"uptime":    time.Now().Sub(startTime).String(),
		}))
	})

	/**
	 * Servers list
	 */
	mux.GetFunc("/servers", func(w http.ResponseWriter, req *http.Request) {
		io.WriteString(w, Marshal(servers))
	})

	/**
	 * Server by name
	 */
	mux.GetFunc("/servers/:name", func(w http.ResponseWriter, req *http.Request) {
		name := bone.GetValue(req, "name")
		io.WriteString(w, Marshal(servers.(map[string]interface{})[name]))
	})

	/**
	 * Server stats
	 */
	mux.GetFunc("/servers/:name/stats", func(w http.ResponseWriter, req *http.Request) {
		name := bone.GetValue(req, "name")
		io.WriteString(w, Marshal(name))
	})

	/* Go listen! */
	http.ListenAndServe(cfg.Bind, mux)
}
