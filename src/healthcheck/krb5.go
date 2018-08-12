/**
 * krb5.go - Kerberos HealthCheck
 *
 * @author Adun Chan <stutiredboy@gmail.com>
 */

package healthcheck

import (
	"../config"
	"../core"
	"../logging"
	"time"
	"strings"
	"context"
	"gopkg.in/jcmturner/gokrb5.v5/client"
	"gopkg.in/jcmturner/gokrb5.v5/keytab"
	krb5config "gopkg.in/jcmturner/gokrb5.v5/config"
)

/**
 * Krb5 healthcheck
 */
func krb5(t core.Target, cfg config.HealthcheckConfig, result chan<- CheckResult) {
	log := logging.For("healthcheck/krb5")

	krb5Timeout, _ := time.ParseDuration(cfg.Timeout)
	krb5Conf, err := krb5config.Load(
		strings.Replace(cfg.Krb5Conf, "%host%", t.Host, -1))
	if err != nil {
		panic(err)
	}
	krb5Realm := cfg.Krb5Realm
	krb5Username := cfg.Krb5Username
	krb5Keytab, err := keytab.Load(cfg.Krb5Keytab)
	if err != nil {
		panic(err)
	}

	checkResult := CheckResult{
		Target: t,
	}

	cl := client.NewClientWithKeytab(krb5Username, krb5Realm, krb5Keytab)
	cl.WithConfig(krb5Conf)

	/*
	 * Kerberos has not native timeout,
	 * use time and context to control the Login time.
	 */
	loginChan := make(chan bool, 1)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func(ctx context.Context) {
		err := cl.Login()
		select {
		case <- ctx.Done():
			log.Debug("drop timeout results for ", t.Host, ":", t.Port, ".")
			close(loginChan)
		default:
			if err != nil {
				log.Warn(err)
				loginChan <- false
			} else {
				log.Debug("Kinit successed with ", t.Host, ":", t.Port, ".")
				loginChan <- true
			}
		}
	}(ctx)

	select {
	case live := <-loginChan :
		checkResult.Live = live
	case <-time.After(krb5Timeout) :
		cancel()
		log.Warn("Kinit timeout with ", t.Host, ":", t.Port, ".")
		checkResult.Live = false
	}

	select {
	case result <- checkResult:
	default:
		log.Warn("Channel is full. Discarding value")
	}

}
