package healthcheck

/**
 * probe.go - TCP/UDP probe healthcheck
 *
 * @author Yousong Zhou <zhouyousong@yunionyun.com>
 * @author Illarion Kovalchuk <illarion.kovalchuk@gmail.com>
 */

import (
	"bytes"
	"io"
	"net"
	"regexp"
	"time"

	"github.com/yyyar/gobetween/config"
	"github.com/yyyar/gobetween/core"
	"github.com/yyyar/gobetween/logging"
)

func probe(t core.Target, cfg config.HealthcheckConfig, result chan<- CheckResult) {
	log := logging.For("healthcheck/probe")

	timeout, _ := time.ParseDuration(cfg.Timeout)

	checkResult := CheckResult{
		Live:   false,
		Target: t,
	}

	defer func() {
		select {
		case result <- checkResult:
		default:
			log.Warn("Channel is full. Discarding value")
		}
	}()

	conn, err := net.DialTimeout(cfg.ProbeProtocol, t.Address(), timeout)
	if err != nil {
		checkResult.Live = false
		return
	}

	defer conn.Close()

	send := []byte(cfg.ProbeSend)

	recv := []byte(cfg.ProbeRecv)
	recvLen := cfg.ProbeRecvLen

	if recvLen == 0 {
		recvLen = len(recv)
	}

	if timeout > 0 {
		err = conn.SetWriteDeadline(time.Now().Add(timeout))
		if err != nil {
			log.Errorf("Could not set write timeout: %v", err)
			return
		}
	}

	n, err := conn.Write(send)
	if err != nil {
		log.Debugf("Could not send probe: %v", err)
		return
	}

	if n != len(send) {
		log.Debugf("Incomplete probe write")
		return
	}

	if timeout > 0 {
		err = conn.SetReadDeadline(time.Now().Add(timeout))
		if err != nil {
			log.Errorf("Could not set read timeout: %v", err)
			return
		}
	}

	actual := make([]byte, recvLen)
	n, err = io.ReadFull(conn, actual)
	if err != nil {
		log.Debugf("Could not read from backend: %v", err)
		return
	}

	switch cfg.ProbeStrategy {
	case "starts_with":
		if !bytes.Equal(actual, recv) {
			log.Debugf("Bytes received from backend:\n% x\nbytes expected:\n% x", actual, recv)
			return
		}
	case "regexp":
		re := regexp.MustCompile(cfg.ProbeRecv)
		if !re.Match(actual) {
			log.Debugf("Bytes received from backend: % x did not match %v", actual, cfg.ProbeRecv)
			return
		}
	default:
		panic("probe_strategy should be checked in manager")
	}

	checkResult.Live = true
}
