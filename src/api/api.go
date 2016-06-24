/**
 * api.go - rest api implementation
 *
 * @author Yaroslav Pogrebnyak <yyyaroslav@gmail.com>
 */
package api

import (
	"../config"
	"../info"
	"../logging"
	"../manager"
	"github.com/gin-gonic/gin"
	"net/http"
	"os"
	"syscall"
	"time"
)

/* gin app */
var app *gin.Engine

/**
 * Initialize module
 */
func init() {
	gin.SetMode(gin.ReleaseMode)
}

/**
 * Starts REST API server
 */
func Start(cfg config.ApiConfig) {

	var log = logging.For("api")

	if !cfg.Enabled {
		log.Info("API disabled")
		return
	}

	log.Info("Starting up API")

	app = gin.New()

	/* -------------------- handlers --------------------- */

	/* ----- globals ----- */

	/**
	 * Global stats
	 */
	app.GET("/", func(c *gin.Context) {

		rusage := syscall.Rusage{}
		syscall.Getrusage(syscall.RUSAGE_SELF, &rusage)

		c.IndentedJSON(http.StatusOK, gin.H{
			"rss":        rusage.Maxrss,
			"pid":        os.Getpid(),
			"time":       time.Now(),
			"startTime":  info.StartTime,
			"uptime":     time.Now().Sub(info.StartTime).String(),
			"version":    info.Version,
			"configPath": info.ConfigPath,
		})
	})

	/* ----- servers ----- */

	app.GET("/servers", func(c *gin.Context) {
		c.IndentedJSON(http.StatusOK, manager.All())
	})

	app.GET("/servers/:name", func(c *gin.Context) {
		name := c.Param("name")
		c.IndentedJSON(http.StatusOK, manager.Get(name))
	})

	app.DELETE("/servers/:name", func(c *gin.Context) {
		name := c.Param("name")
		manager.Delete(name)
		c.IndentedJSON(http.StatusOK, nil)
	})

	app.POST("/servers/:name", func(c *gin.Context) {

		name := c.Param("name")

		cfg := config.Server{}
		if err := c.BindJSON(&cfg); err != nil {
			c.IndentedJSON(http.StatusBadRequest, err.Error())
			return
		}

		if err := manager.Create(name, cfg); err != nil {
			c.IndentedJSON(http.StatusConflict, err.Error())
			return
		}

		c.IndentedJSON(http.StatusOK, nil)
	})

	app.GET("/servers/:name/stats", func(c *gin.Context) {
		name := c.Param("name")
		c.IndentedJSON(http.StatusOK, manager.Stats(name))
	})

	app.Run(cfg.Bind)
}
