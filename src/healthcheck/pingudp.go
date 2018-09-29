/**
 * pingudp.go - UDP ping healthcheck
 *
 * @author Yousong Zhou <zhouyousong@yunionyun.com>
 */

package healthcheck

import (
	"net"
	"time"

	"../config"
	"../core"
	"../logging"
)

const defaultUDPTimeout = 5 * time.Second

// Check executes a UDP healthcheck.
func pingudp(t core.Target, cfg config.HealthcheckConfig, result chan<- CheckResult) {
	live := false
	log := logging.For("healthcheck/pingUdp")
	defer func() {
		checkResult := CheckResult{
			Target: t,
			Live:   live,
		}
		select {
		case result <- checkResult:
		default:
			log.Warn("Channel is full. Discarding value")
		}
	}()

	addr := t.Host + ":" + t.Port
	conn, err := net.Dial("udp", addr)
	if err != nil {
		return
	}
	defer conn.Close()

	timeout, _ := time.ParseDuration(cfg.Timeout)
	if timeout == time.Duration(0) {
		timeout = defaultUDPTimeout
	}
	deadline := time.Now().Add(timeout)
	err = conn.SetDeadline(deadline)
	if err != nil {
		return
	}

	udpConn := conn.(*net.UDPConn)
	if _, err = udpConn.Write([]byte(cfg.Send)); err != nil {
		return
	}

	buf := make([]byte, len(cfg.Receive))
	n, _, err := udpConn.ReadFrom(buf)
	if err != nil {
		return
	}

	got := string(buf[0:n])
	if got != cfg.Receive {
		return
	}
	live = true
}
