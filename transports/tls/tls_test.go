package tls

import (
	"bytes"
	"crypto/tls"
	"os"
	"testing"
)

const (
	// these constants may need to change when testdata content changes
	caRootCertFileLocation = "./testdata/ca_root.pem"
	caRootCertSubjectCN    = "linuxctl ECC Root Certification Authority (Test)"
	caIntCertFileLocation  = "./testdata/ca_int.pem"
	caIntCertSubjectCN     = "linuxctl ECC Intermediate Certification Authority (Test)"
	clientCertFileLocation = "./testdata/client_logspoutClient.pem"
	clientKeyFileLocation  = "./testdata/client_logspoutClient-key.pem"
)

// helper function to create TLS config while handling error
func createtestTLSConfig(t *testing.T) *tls.Config {
	testTLSConfig, err := createTLSConfig()
	if err != nil {
		t.Fatalf("we got an unexpected error while calling createTLSConfig: %s", err)
	}

	return testTLSConfig
}

// TestEmptyTrustStore should test the behaviour of having
// an empty TLS CA trust store.
func TestEmptyTrustStore(t *testing.T) {

	os.Setenv("LOGSPOUT_TLS_DISABLE_SYSTEM_ROOTS", "true")
	os.Unsetenv("LOGSPOUT_TLS_CA_CERTS")

	testTLSConfig := createtestTLSConfig(t)

	numOfTrustedCerts := len(testTLSConfig.RootCAs.Subjects())
	if numOfTrustedCerts != 0 {
		t.Fatalf("expected 0 RootCAs but got: %d", numOfTrustedCerts)
	}

}

// TestSingleCustomCA should test the behaviour of loading
// a single custom CA certificate in to the trust store.
func TestSingleCustomCA(t *testing.T) {

	os.Setenv("LOGSPOUT_TLS_DISABLE_SYSTEM_ROOTS", "true")
	os.Setenv("LOGSPOUT_TLS_CA_CERTS", caRootCertFileLocation)

	testTLSConfig := createtestTLSConfig(t)

	// check if trust store has this cert
	if !bytes.Contains(testTLSConfig.RootCAs.Subjects()[0], []byte(caRootCertSubjectCN)) {
		t.Errorf("failed to load custom root CA into trust store: %s", caRootCertFileLocation)
	}

}

// TestMultipleCustomCAs should test the behaviour of loading
// multiple custom CA certificates in to the trust store.
func TestMultipleCustomCAs(t *testing.T) {

	os.Setenv("LOGSPOUT_TLS_DISABLE_SYSTEM_ROOTS", "true")
	os.Setenv("LOGSPOUT_TLS_CA_CERTS", caRootCertFileLocation+","+caIntCertFileLocation)

	testTLSConfig := createtestTLSConfig(t)

	// check that both certificates are in the trust store
	if !bytes.Contains(testTLSConfig.RootCAs.Subjects()[0], []byte(caRootCertSubjectCN)) {
		t.Errorf("failed to load custom root CA into trust store: %s", caRootCertFileLocation)
	}
	if !bytes.Contains(testTLSConfig.RootCAs.Subjects()[1], []byte(caIntCertSubjectCN)) {
		t.Errorf("failed to load custom intermediate CA into trust store: %s", caIntCertFileLocation)
	}
}

// TestSystemRootCAs should test that by default we load the system trust store
func TestSystemRootCAs(t *testing.T) {

	// default behaviour is none of these environment variables are set
	os.Unsetenv("LOGSPOUT_TLS_DISABLE_SYSTEM_ROOTS")
	os.Unsetenv("LOGSPOUT_TLS_CA_CERTS")

	testTLSConfig := createtestTLSConfig(t)
	// its possible that the system does not have a trust store (minimal docker container for example)
	if len(testTLSConfig.RootCAs.Subjects()) < 1 {
		t.Errorf("after loading system trust store we still have 0. Do you have a system trust store?")
	}

}

// TestSystemRootCAsAndCustomCAs should test that we can load
// both system CAs and custom CAs into trust store
func TestSystemRootCAsAndCustomCAs(t *testing.T) {

	os.Unsetenv("LOGSPOUT_TLS_DISABLE_SYSTEM_ROOTS")
	os.Unsetenv("LOGSPOUT_TLS_CA_CERTS")
	testTLSConfig := createtestTLSConfig(t)
	systemCACount := len(testTLSConfig.RootCAs.Subjects())

	os.Setenv("LOGSPOUT_TLS_CA_CERTS", caRootCertFileLocation)
	testTLSConfig = createtestTLSConfig(t)
	currentCACount := len(testTLSConfig.RootCAs.Subjects())
	if currentCACount != (systemCACount + 1) {
		t.Errorf("expected %d certs in trust store but got %d", systemCACount+1, currentCACount)
	}
}

// TestLoadingClientCert should test the behaviour of loading
// a pem encoded client x509 certificate and private key
func TestLoadingClientCertAndKey(t *testing.T) {

	os.Unsetenv("LOGSPOUT_TLS_DISABLE_SYSTEM_ROOTS")
	os.Unsetenv("LOGSPOUT_TLS_CA_CERTS")
	os.Setenv("LOGSPOUT_TLS_CLIENT_CERT", clientCertFileLocation)
	os.Setenv("LOGSPOUT_TLS_CLIENT_KEY", clientKeyFileLocation)

	testTLSConfig := createtestTLSConfig(t)
	if len(testTLSConfig.Certificates) < 1 {
		t.Error("failed to load client certficate and key")
	}
}
