package healthcheck

import (
	"context"
	"github.com/yyyar/gobetween/config"
	"github.com/yyyar/gobetween/core"
	"github.com/yyyar/gobetween/logging"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

func httpCheck(t core.Target, cfg config.HealthcheckConfig, result chan<- CheckResult) {
	log := logging.For("healthcheck/http")

	timeout, _ := time.ParseDuration(cfg.Timeout)
	timeoutCtxt, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

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

	client := &http.Client{}

	targetUrl, err := url.ParseRequestURI(cfg.HttpPath)
	if err != nil {
		log.Warnf("Unable to parse http_path in healthcheck config: %s", err)
		return
	}
	if cfg.HttpPort > 0 {
		targetUrl.Host = t.Host + ":" + strconv.Itoa(cfg.HttpPort)
	} else {
		targetUrl.Host = t.Host + ":" + t.Port
	}
	if targetUrl.Scheme == "" {
		targetUrl.Scheme = "http"
	}

	method := cfg.HttpMethod
	if method == "" {
		method = "GET"
	}

	req, err := http.NewRequestWithContext(timeoutCtxt, method, targetUrl.String(), nil)
	if err != nil {
		log.Debugf("Could not send healthcheck request to %s: %v", targetUrl, err)
		return
	}

	response, err := client.Do(req)
	if err != nil {
		log.Debugf("Could not send healthcheck request to %s: %v", t, err)
		return
	}

	if response.StatusCode == 200 {
		checkResult.Live = true
	} else {
		log.Debugf("Failed healthcheck from %v, received status %s", t, err)
	}
}
