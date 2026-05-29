package session

import "time"

type Config struct {
	MaxRequests        uint64
	MaxResponses       uint64
	ClientIdleTimeout  time.Duration
	BackendIdleTimeout time.Duration
	Transparent        bool
}
