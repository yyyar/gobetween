package tls

/**
 * tls.go - Tls mapping utils
 *
 * @author Yaroslav Pogrebnyak <yyyaroslav@gmail.com>
 */

import (
	"crypto/tls"
	"crypto/x509"
	"io/ioutil"

	"github.com/yyyar/gobetween/config"
)

/**
 * TLS Ciphers mapping
 */
var suites map[string]uint16 = map[string]uint16{
	"TLS_RSA_WITH_RC4_128_SHA":                tls.TLS_RSA_WITH_RC4_128_SHA,
	"TLS_RSA_WITH_3DES_EDE_CBC_SHA":           tls.TLS_RSA_WITH_3DES_EDE_CBC_SHA,
	"TLS_RSA_WITH_AES_128_CBC_SHA":            tls.TLS_RSA_WITH_AES_128_CBC_SHA,
	"TLS_RSA_WITH_AES_256_CBC_SHA":            tls.TLS_RSA_WITH_AES_256_CBC_SHA,
	"TLS_RSA_WITH_AES_128_GCM_SHA256":         tls.TLS_RSA_WITH_AES_128_GCM_SHA256,
	"TLS_RSA_WITH_AES_256_GCM_SHA384":         tls.TLS_RSA_WITH_AES_256_GCM_SHA384,
	"TLS_ECDHE_ECDSA_WITH_RC4_128_SHA":        tls.TLS_ECDHE_ECDSA_WITH_RC4_128_SHA,
	"TLS_ECDHE_ECDSA_WITH_AES_128_CBC_SHA":    tls.TLS_ECDHE_ECDSA_WITH_AES_128_CBC_SHA,
	"TLS_ECDHE_ECDSA_WITH_AES_256_CBC_SHA":    tls.TLS_ECDHE_ECDSA_WITH_AES_256_CBC_SHA,
	"TLS_ECDHE_RSA_WITH_RC4_128_SHA":          tls.TLS_ECDHE_RSA_WITH_RC4_128_SHA,
	"TLS_ECDHE_RSA_WITH_3DES_EDE_CBC_SHA":     tls.TLS_ECDHE_RSA_WITH_3DES_EDE_CBC_SHA,
	"TLS_ECDHE_RSA_WITH_AES_128_CBC_SHA":      tls.TLS_ECDHE_RSA_WITH_AES_128_CBC_SHA,
	"TLS_ECDHE_RSA_WITH_AES_256_CBC_SHA":      tls.TLS_ECDHE_RSA_WITH_AES_256_CBC_SHA,
	"TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256":   tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
	"TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256": tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,
	"TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384":   tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
	"TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384": tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,
}

/**
 * TLS Versions mappings
 */
var versions map[string]uint16 = map[string]uint16{
	"ssl3":   tls.VersionSSL30,
	"tls1":   tls.VersionTLS10,
	"tls1.1": tls.VersionTLS11,
	"tls1.2": tls.VersionTLS12,
}

/**
 * Maps tls version from string to golang constant
 */
func MapVersion(version string) uint16 {
	return versions[version]
}

/**
 * Maps tls ciphers from array of strings to array of golang constants
 */
func MapCiphers(ciphers []string) []uint16 {

	if ciphers == nil || len(ciphers) == 0 {
		return nil
	}

	result := []uint16{}

	for _, s := range ciphers {
		c := suites[s]
		if c == 0 {
			continue
		}
		result = append(result, c)
	}

	return result
}

func MakeTlsConfig(tlsC *config.Tls, getCertificate func(*tls.ClientHelloInfo) (*tls.Certificate, error)) (*tls.Config, error) {

	if tlsC == nil {
		return nil, nil
	}

	tlsConfig := &tls.Config{}

	tlsConfig.CipherSuites = MapCiphers(tlsC.Ciphers)
	tlsConfig.PreferServerCipherSuites = tlsC.PreferServerCiphers
	tlsConfig.MinVersion = MapVersion(tlsC.MinVersion)
	tlsConfig.MaxVersion = MapVersion(tlsC.MaxVersion)
	tlsConfig.SessionTicketsDisabled = !tlsC.SessionTickets

	if getCertificate != nil {
		tlsConfig.GetCertificate = getCertificate
		return tlsConfig, nil
	}

	var crt tls.Certificate
	var err error
	if crt, err = tls.LoadX509KeyPair(tlsC.CertPath, tlsC.KeyPath); err != nil {
		return nil, err
	}

	tlsConfig.Certificates = []tls.Certificate{crt}

	return tlsConfig, nil
}

/**
 * MakeBackendTLSConfig makes a tls.Config for connecting to backends
 */
func MakeBackendTLSConfig(backendsTls *config.BackendsTls) (*tls.Config, error) {

	if backendsTls == nil {
		return nil, nil
	}

	var err error

	result := &tls.Config{
		InsecureSkipVerify:       backendsTls.IgnoreVerify,
		CipherSuites:             MapCiphers(backendsTls.Ciphers),
		PreferServerCipherSuites: backendsTls.PreferServerCiphers,
		MinVersion:               MapVersion(backendsTls.MinVersion),
		MaxVersion:               MapVersion(backendsTls.MaxVersion),
		SessionTicketsDisabled:   !backendsTls.SessionTickets,
	}
	
//Sometimes we will encounter a certificate and key that require a password for parsing, 
//in this case we must first use IsEncryptedPEMBlock()
//
//Next, if IsEncryptedPEMBlock() returns true, we whould use decryptPEMBlock 
// with the cert object and the password used to encrypt it and return a slice of decrypted DER encoded bytes
//now, we can use LoadX509KeyPair with the decrypted DER encoded bytes (passed as the decrypted cert) and encrypt the tls

//start region
	if backendsTls.CertPath != nil && backendsTls.KeyPath != nil {
		var password := backendTls.CertPass
		var crt tls.Certificate
		var crtDcrypt;
		if IsEncryptedPEMBlock(backendsTls.CertPath) {
			crtDcrypt = decryptPEMBlock(backendsTls.CertPath, password)
			
			if crt, err = tls.LoadX509KeyPair(*crtDcrypt, *backendsTls.KeyPath); err != nil {
				return nil, err
		}
			}
		result.Certificates = []tls.Certificate{crt}
	}else{
		if crt, err = tls.LoadX509KeyPair(*crtDcrypt, *backendsTls.KeyPath); err != nil {
				return nil, err
		}
			}
		result.Certificates = []tls.Certificate{crt}
	}
//end region


	if backendsTls.RootCaCertPath != nil {

		var caCertPem []byte

		if caCertPem, err = ioutil.ReadFile(*backendsTls.RootCaCertPath); err != nil {
			return nil, err
		}

		caCertPool := x509.NewCertPool()
		if ok := caCertPool.AppendCertsFromPEM(caCertPem); !ok {
			return nil, err
		}

		result.RootCAs = caCertPool

	}

	return result, nil

}
