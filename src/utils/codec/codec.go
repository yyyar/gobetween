/**
 * codec.go - decoding utils
 *
 * @author Yaroslav Pogrebnyak <yyyaroslav@gmail.com>
 */
package codec

import (
	"encoding/json"
	"errors"
	"github.com/BurntSushi/toml"
)

/**
 * Decode data based on format
 * Currently supported: toml and json
 */
func Decode(data string, out interface{}, format string) error {

	switch format {
	case "toml":
		_, err := toml.Decode(data, out)
		return err
	case "json":
		return json.Unmarshal([]byte(data), out)
	default:
		return errors.New("Unknown format " + format)
	}
}
