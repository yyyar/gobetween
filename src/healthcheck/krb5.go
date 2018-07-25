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
	"fmt"
	"gopkg.in/jcmturner/gokrb5.v5/client"
	"gopkg.in/jcmturner/gokrb5.v5/keytab"
	krb5config "gopkg.in/jcmturner/gokrb5.v5/config"
)

/**
 * Krb5 healthcheck
 */
func krb5(t core.Target, cfg config.HealthcheckConfig, result chan<- CheckResult) {
	log := logging.For("healthcheck/krb5")

	krb5Conf, err := krb5config.Load(fmt.Sprintf("%s/krb5.%s.conf", cfg.Krb5Conf, t.Host))
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
	err = cl.Login()
	if err != nil {
		checkResult.Live = false
		log.Warn(err)
	} else {
		checkResult.Live = true
		log.Info("Kinit successed with ", t.Host, ":", t.Port, ".")
	}

	select {
	case result <- checkResult:
	default:
		log.Warn("Channel is full. Discarding value")
	}

}
