## HealthCheck For Kerberos V5

Example for gobetween.toml:

```
[servers.krb5]
bind = "0.0.0.0:88"
protocol = "udp"
  [servers.krb5.udp]
  max_responses = 0
  [servers.krb5.discovery]
  kind = "static"
  static_list = [
    "kdc01.chenxiaosheng.com:88",
    "10.176.42.204:88",
  ]
  [servers.krb5.healthcheck]
  kind = "krb5"
  interval = "10s"
  timeout = "5s"
  fails = 2
  passes = 3
  krb5_conf = "/home/gobetween/etc/kdc"
  krb5_realm = "CHENXIAOSHENG.COM"
  krb5_username = "kdc.healthcheck"
  krb5_keytab = "/home/gobetween/etc/kdc/kdc.healthcheck.keytab"
```

### kind

For Kerberos/KDC health check, you should use string `krb5` always.

### krb5_conf

Directory for locate [KRB5_CONFIG](http://web.mit.edu/kerberos/krb5-latest/doc/admin/env_variables.html) to check Kerberos/KDC health.

1. One host shoud have one config file here, named as: `krb5.${host}.conf`, for example above ,shoud have: `krb5.kdc01.chenxiaosheng.com.conf` and `krb5.10.176.42.204.conf`;
2. More about kerberos's configure, read [here](http://web.mit.edu/kerberos/krb5-1.12/doc/admin/conf_files/krb5_conf.html) please;

### krb5_realm

[Realm name](http://web.mit.edu/kerberos/krb5-latest/doc/admin/realm_config.html#realm-name)

### krb5_username

[Kerberos Principal](http://web.mit.edu/kerberos/krb5-1.5/krb5-1.5.4/doc/krb5-user/What-is-a-Kerberos-Principal_003f.html)

### krb5_keytab

[The Keytab File](http://web.mit.edu/Kerberos/krb5-1.5/krb5-1.5.3/doc/krb5-install/The-Keytab-File.html)
