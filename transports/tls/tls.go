// +build go1.8

package tls

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"os"
	"strings"

	"github.com/gliderlabs/logspout/adapters/raw"
	"github.com/gliderlabs/logspout/router"
)

// package wide cache of TLS config
var clientTLSConfig *tls.Config

const (
	// constants used to identify environment variable names
	envDisableSystemRoots = "LOGSPOUT_TLS_DISABLE_SYSTEM_ROOTS"
	envCaCerts            = "LOGSPOUT_TLS_CA_CERTS"
	envClientCert         = "LOGSPOUT_TLS_CLIENT_CERT"
	envClientKey          = "LOGSPOUT_TLS_CLIENT_KEY"
	envTLSHardening       = "LOGSPOUT_TLS_HARDENING"
)

func init() {
	router.AdapterTransports.Register(new(tlsTransport), "tls")
	// convenience adapters around raw adapter
	router.AdapterFactories.Register(rawTLSAdapter, "tls")

	// we should load our TLS configuration only once
	// since it is not expected to change during runtime
	var err error
	clientTLSConfig, err = createTLSConfig()

	// without a valid/desired TLS config, we should exit
	if err != nil {
		log.Fatalf("error with TLSConfig: %s", err)
	}

}

func rawTLSAdapter(route *router.Route) (router.LogAdapter, error) {
	route.Adapter = "raw+tls"
	return raw.NewRawAdapter(route)
}

type tlsTransport int

func (t *tlsTransport) Dial(addr string, options map[string]string) (net.Conn, error) {

	// at this point, if our trust store is empty, there is no point of continuing...
	if len(clientTLSConfig.RootCAs.Subjects()) < 1 {
		return nil, fmt.Errorf("FATAL: TLS CA trust store is empty! Can not trust any TLS endpoints: tls://%s", addr)
	}

	// attempt to establish the TLS connection
	conn, err := tls.Dial("tcp", addr, clientTLSConfig)
	if err != nil {
		return nil, err
	}
	return conn, nil
}

// createTLSConfig creates the required TLS configuration that we need to establish a TLS connection
func createTLSConfig() (*tls.Config, error) {

	var err error
	tlsConfig := &tls.Config{}

	// use stronger TLS settings if enabled
	// TODO: perhaps this should be default setting
	if os.Getenv(envTLSHardening) != "true" {
		tlsConfig.MinVersion = tls.VersionTLS11
		tlsConfig.InsecureSkipVerify = false
		// allowed ciphers. Disable CBC suites (Lucky13). For now we allow RSA
		tlsConfig.CipherSuites = []uint16{
			tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,
			tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
			tls.TLS_RSA_WITH_AES_256_GCM_SHA384,
		}
		// curve reference: http://safecurves.cr.yp.to/
		tlsConfig.CurvePreferences = []tls.CurveID{
			// this curve is a non-NIST curve with no NSA influence. Prefer this over all others!
			tls.X25519,
			// These curves are provided by NIST; prefer in descending order
			tls.CurveP521,
			tls.CurveP384,
			tls.CurveP256,
		}
	}

	// load possible TLS CA chain(s) for server certificate validation
	// starting with an empty pool
	tlsConfig.RootCAs = x509.NewCertPool()

	// load system root CA trust store by default, unless configured not to
	// if we cannot, then it's fatal.
	// NOTE that we ONLY fail if SystemCertPool returns an error,
	// not if our system trust store is empty or doesn't exist!
	if os.Getenv(envDisableSystemRoots) != "true" {
		tlsConfig.RootCAs, err = x509.SystemCertPool()
		if err != nil {
			return nil, err
		}
	}

	// load custom certificates specified by configuration:
	// we expect a comma separated list of certificate file paths
	// if we fail to load a certificate, we should treat this to be fatal
	// as the user may not wish to send logs through an untrusted TLS connection
	// also note that each file specified above can contain one or more certificates
	// and we also _DO NOT_ check if they are CA certificates (in case of self-signed)
	if certsEnv := os.Getenv(envCaCerts); certsEnv != "" {
		certFilePaths := strings.Split(certsEnv, ",")

		for _, certFilePath := range certFilePaths {
			// each pem file may contain more than one certficate
			certBytes, err := ioutil.ReadFile(certFilePath)
			if err != nil {
				return nil, err
			}
			if !tlsConfig.RootCAs.AppendCertsFromPEM(certBytes) {
				return nil, fmt.Errorf("failed to load CA certificate(s): %s", certFilePath)
			}
		}
	}

	// load a client certificate and key if enabled
	// we should fail if unable to load the keypair since the user intended mutual authentication
	clientCertFilePath := os.Getenv(envClientCert)
	clientKeyFilePath := os.Getenv(envClientKey)
	if clientCertFilePath != "" && clientKeyFilePath != "" {
		clientCert, err := tls.LoadX509KeyPair(clientCertFilePath, clientKeyFilePath)
		if err != nil {
			return nil, err
		}
		// according to TLS spec, the client _SHOULD_ send the CA certificate chain
		// which issued its own client cert (at the very least the intermediates).
		// However, we will make this optional as the client cert pem file can contain more than one certificate
		tlsConfig.Certificates = []tls.Certificate{clientCert}
	}

	return tlsConfig, nil

}
