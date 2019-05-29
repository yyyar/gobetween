package utils

/**
 * env.go - env vars helpers
 *
 * @author Yaroslav Pogrebnyak <yyyaroslav@gmail.com>
 */

import (
	"os"
	"regexp"
	"strings"
)

//
// SubstituteEnvVars replaces placeholders ${...} with env var value
//
func SubstituteEnvVars(data string) string {

	var re = regexp.MustCompile(`\${.*?}`)

	vars := re.FindAllString(data, -1)
	for _, v := range vars {
		data = strings.ReplaceAll(data, v, os.Getenv(v[2:len(v)-1]))
	}
	return data
}
