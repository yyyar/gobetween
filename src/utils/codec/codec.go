package codec

/**
 * codec.go - decoding utils
 *
 * @author Yaroslav Pogrebnyak <yyyaroslav@gmail.com>
 */

import (
	"bytes"
	"encoding/json"
	"errors"

	"github.com/burntsushi/toml"
)

/**
 * Encode data based on format
 * Currently supported: toml and json
 */
func Encode(in interface{}, out *string, format string) error {

	switch format {
	case "toml":
		buf := new(bytes.Buffer)
		if err := toml.NewEncoder(buf).Encode(in); err != nil {
			return err
		}
		*out = buf.String()
		return nil
	case "json":
		buf, err := json.MarshalIndent(in, "", "    ")
		if err != nil {
			return err
		}
		*out = string(buf)
		return nil
	default:
		return errors.New("Unknown format " + format)
	}
}

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
