/**
 * public.go - / rest api implementation
 *
 * @author Mike Schroeder <m.schroeder223@gmail.com>
 */
package api

import (
	"github.com/gin-gonic/gin"
	"net/http"
)

/**
 * Attaches / handlers
 */
func attachPublic(app *gin.RouterGroup) {

	/**
	 * Simple 200 and OK response
	 */
	app.GET("/ping", func(c *gin.Context) {
		c.String(http.StatusOK, "OK")
	})
}
