/**
 * root.go - / rest api implementation
 *
 * @author Yaroslav Pogrebnyak <yyyaroslav@gmail.com>
 */
package api

import (
	"../info"
	"../manager"
	"github.com/gin-gonic/gin"
	"net/http"
	"os"
	"time"
)

/**
 * Attaches / handlers
 */
func attachRoot(app *gin.RouterGroup) {

	/**
	 * Global stats
	 */
	app.GET("/", func(c *gin.Context) {

		c.IndentedJSON(http.StatusOK, gin.H{
			"pid":       os.Getpid(),
			"time":      time.Now(),
			"startTime": info.StartTime,
			"uptime":    time.Now().Sub(info.StartTime).String(),
			"version":   info.Version,
		})
	})

	/**
	 * Dump current config as TOML
	 */
	app.GET("/dump", func(c *gin.Context) {
		txt, err := manager.DumpConfig()
		if err != nil {
			c.IndentedJSON(http.StatusInternalServerError, err)
			return
		}
		c.String(http.StatusOK, txt)
	})
}
