package utils

/**
 * time.go - Time utils
 *
 * @author Yaroslav Pogrebnyak <yyyaroslav@gmail.com>
 */

import (
	"time"
)

/**
 * Parse duration or return default
 */
func ParseDurationOrDefault(s string, defaultDuration time.Duration) time.Duration {

	var d time.Duration
	var err error

	if s == "" {
		return defaultDuration
	}

	d, err = time.ParseDuration(s)
	if err != nil {
		return defaultDuration
	}

	return d
}
