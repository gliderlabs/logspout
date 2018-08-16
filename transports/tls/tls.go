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

const (
	// constants used to identify environment variable names
	envDisableSystemRoots = "LOGSPOUT_TLS_DISABLE_SYSTEM_ROOTS"
	envCaCerts            = "LOGSPOUT_TLS_CA_CERTS"
	envClientCert         = "LOGSPOUT_TLS_CLIENT_CERT"
	envClientKey          = "LOGSPOUT_TLS_CLIENT_KEY"
	envTLSHardening       = "LOGSPOUT_TLS_HARDENING"
)

var (
	// package wide cache of TLS config
	clientTLSConfig *tls.Config
	// PCI compliance as of Jun 30, 2018: anything under TLS 1.1 must be disabled
	// we bump this up to TLS 1.2 so we can support best possible ciphers
	hardenedMinVersion = uint16(tls.VersionTLS12)
	// allowed ciphers when in hardened mode
	// disable CBC suites (Lucky13 attack) this means TLS 1.1 can't work (no GCM)
	// only use perfect forward secrecy ciphers
	hardenedCiphers = []uint16{
		tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,
		tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
		tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,
		tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
		// these ciphers require go 1.8+
		tls.TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305,
		tls.TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305,
	}
	// EC curve preference when in hardened mode
	// curve reference: http://safecurves.cr.yp.to/
	hardenedCurvePreferences = []tls.CurveID{
		// this curve is a non-NIST curve with no NSA influence. Prefer this over all others!
		// this curve required go 1.8+
		tls.X25519,
		// These curves are provided by NIST; prefer in descending order
		tls.CurveP521,
		tls.CurveP384,
		tls.CurveP256,
	}
)

type tlsTransport int

func init() {
	router.AdapterTransports.Register(new(tlsTransport), "tls")
	// convenience adapters around raw adapter
	router.AdapterFactories.Register(rawTLSAdapter, "tls")

	// we should load our TLS configuration only once
	// since it is not expected to change during runtime
	var err error
	if clientTLSConfig, err = createTLSConfig(); err != nil {
		// without a valid/desired TLS config, we should exit
		log.Fatalf("error with TLSConfig: %s", err)
	}
}

func rawTLSAdapter(route *router.Route) (r router.LogAdapter, err error) {
	route.Adapter = "raw+tls"
	r, err = raw.NewRawAdapter(route)
	return
}

func (t *tlsTransport) Dial(addr string, options map[string]string) (conn net.Conn, err error) {
	// at this point, if our trust store is empty, there is no point of continuing
	// since it would be impossible to successfully validate any x509 server certificates
	if len(clientTLSConfig.RootCAs.Subjects()) < 1 {
		err = fmt.Errorf("FATAL: TLS CA trust store is empty! Can not trust any TLS endpoints: tls://%s", addr)
		return
	}

	// attempt to establish the TLS connection
	conn, err = tls.Dial("tcp", addr, clientTLSConfig)
	return
}

// createTLSConfig creates the required TLS configuration that we need to establish a TLS connection
func createTLSConfig() (tlsConfig *tls.Config, err error) {
	tlsConfig = &tls.Config{}

	// use stronger TLS settings if enabled
	// TODO: perhaps this should be default setting
	if os.Getenv(envTLSHardening) == "true" {
		tlsConfig.InsecureSkipVerify = false
		tlsConfig.MinVersion = hardenedMinVersion
		tlsConfig.CipherSuites = hardenedCiphers
		tlsConfig.CurvePreferences = hardenedCurvePreferences
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
			return
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
			var certBytes []byte
			certBytes, err = ioutil.ReadFile(certFilePath)
			if err != nil {
				return
			}
			if !tlsConfig.RootCAs.AppendCertsFromPEM(certBytes) {
				err = fmt.Errorf("failed to load CA certificate(s): %s", certFilePath)
				return
			}
		}
	}

	// load a client certificate and key if enabled
	// we should only attempt this if BOTH cert and key are defined
	clientCertFilePath := os.Getenv(envClientCert)
	clientKeyFilePath := os.Getenv(envClientKey)
	if clientCertFilePath != "" && clientKeyFilePath != "" {
		var clientCert tls.Certificate
		clientCert, err = tls.LoadX509KeyPair(clientCertFilePath, clientKeyFilePath)
		// we should fail if unable to load the keypair since the user intended mutual authentication
		if err != nil {
			return
		}
		// according to TLS spec (RFC 5246 appendix F.1.1) the certificate message
		// must provide a valid certificate chain leading to an acceptable certificate authority.
		// We will make this optional; the client cert pem file can contain more than one certificate
		tlsConfig.Certificates = []tls.Certificate{clientCert}
	}
	return
}
