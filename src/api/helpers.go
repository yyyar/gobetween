/**
 * helpers.go - rest api helpers
 *
 * @author Yaroslav Pogrebnyak <yyyaroslav@gmail.com>
 */

package api

import (
	"bytes"
	"encoding/json"
)

/**
 * Marshal data to pretty-formatter json
 */
func Marshal(data interface{}) string {
	b, _ := json.Marshal(data)
	var out bytes.Buffer
	json.Indent(&out, b, "", "    ")
	return out.String()
}
