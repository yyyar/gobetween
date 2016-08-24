package healthcheck

import (
	"../config"
	"../core"
	"../logging"
	"encoding/hex"
	"net"
	"regexp"
	"strings"
	"time"
)

/**
 * Pattern healthcheck
 */
func udp(t core.Target, cfg config.HealthcheckConfig, result chan<- CheckResult) {
	patternPingTimeoutDuration, _ := time.ParseDuration(cfg.Timeout)

	log := logging.For("healthcheck/pattern")

	checkResult := CheckResult{
		Target: t,
	}

	conn, err := net.Dial("udp", t.Address())

	if err != nil {
		log.Error("Networking error", err)
		conn.Close()
		return
	}

	conn.SetReadDeadline(time.Now().Add(patternPingTimeoutDuration))
	sendbuf, _ := hex.DecodeString(strings.Replace(cfg.SendPattern, " ", "", -1))

	recvbuf := make([]byte, 1024)

	conn.Write(sendbuf)
	n, err := conn.Read(recvbuf)
	conn.Close()

	if err != nil {
		checkResult.Live = false
	} else {
		if cfg.ExpectedPattern == nil {
			checkResult.Live = true
		} else {
			recvStr := hex.EncodeToString(recvbuf[0:n])
			matched, _ := regexp.MatchString(strings.Replace(*cfg.ExpectedPattern, " ", "", -1), recvStr)
			checkResult.Live = matched
		}
	}

	select {
	case result <- checkResult:
	default:
		log.Warn("Channel is full. Discarding value")
	}
}
