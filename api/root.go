package api

/**
 * root.go - / rest api implementation
 *
 * @author Yaroslav Pogrebnyak <yyyaroslav@gmail.com>
 */

import (
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/yyyar/gobetween/info"
	"github.com/yyyar/gobetween/manager"
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
			"pid":           os.Getpid(),
			"time":          time.Now(),
			"startTime":     info.StartTime,
			"uptime":        time.Now().Sub(info.StartTime).String(),
			"version":       info.Version,
			"configuration": info.Configuration,
		})
	})

	/**
	 * Dump current config as TOML
	 */
	app.GET("/dump", func(c *gin.Context) {
		format := c.DefaultQuery("format", "toml")

		data, err := manager.DumpConfig(format)
		if err != nil {
			c.IndentedJSON(http.StatusInternalServerError, err.Error())
			return
		}

		c.String(http.StatusOK, data)
	})
}
