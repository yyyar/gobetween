/**
 * udp.go - UDP ping healthcheck
 *
 * @author Illarion Kovalchuk
 */

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

const (
	UDP_RECV_BUFFER_SIZE = 1024
)

/**
 * Pattern healthcheck
 */
func udp(t core.Target, cfg config.HealthcheckConfig, result chan<- CheckResult) {

	log := logging.For("healthcheck/udp")

	timeout, _ := time.ParseDuration(cfg.Timeout)
	conn, err := net.Dial("udp", t.Address())

	if err != nil {
		log.Error("Networking error", err)
		return
	}

	conn.SetReadDeadline(time.Now().Add(timeout))

	sendbuf, _ := hex.DecodeString(strings.Replace(cfg.UdpSendPattern, " ", "", -1))
	recvbuf := make([]byte, UDP_RECV_BUFFER_SIZE)

	conn.Write(sendbuf)
	n, err := conn.Read(recvbuf)
	conn.Close()

	checkResult := CheckResult{
		Target: t,
	}

	if err != nil {
		checkResult.Live = false
	} else {
		if cfg.UdpExpectedPattern == nil {
			checkResult.Live = true
		} else {
			recvStr := hex.EncodeToString(recvbuf[0:n])
			matched, _ := regexp.MatchString(strings.Replace(*cfg.UdpExpectedPattern, " ", "", -1), recvStr)
			checkResult.Live = matched
		}
	}

	select {
	case result <- checkResult:
	default:
		log.Warn("Channel is full. Discarding value")
	}
}
