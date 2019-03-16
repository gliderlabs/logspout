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
func createTestTLSConfig(t *testing.T) *tls.Config {
	testTLSConfig, err := createTLSConfig()
	if err != nil {
		t.Fatalf("we got an unexpected error while calling createTLSConfig: %s", err)
	}

	return testTLSConfig
}

// TestEmptyTrustStore should test the behaviour of having
// an empty TLS CA trust store.
func TestEmptyTrustStore(t *testing.T) {
	os.Setenv(envDisableSystemRoots, "true")
	os.Unsetenv(envCaCerts)
	testTLSConfig := createTestTLSConfig(t)

	numOfTrustedCerts := len(testTLSConfig.RootCAs.Subjects())
	if numOfTrustedCerts != 0 {
		t.Fatalf("expected 0 RootCAs but got: %d", numOfTrustedCerts)
	}

}

// TestSingleCustomCA should test the behaviour of loading
// a single custom CA certificate in to the trust store.
func TestSingleCustomCA(t *testing.T) {
	os.Setenv(envDisableSystemRoots, "true")
	os.Setenv(envCaCerts, caRootCertFileLocation)
	testTLSConfig := createTestTLSConfig(t)

	// check if trust store has this cert
	if !bytes.Contains(testTLSConfig.RootCAs.Subjects()[0], []byte(caRootCertSubjectCN)) {
		t.Errorf("failed to load custom root CA into trust store: %s", caRootCertFileLocation)
	}

}

// TestMultipleCustomCAs should test the behaviour of loading
// multiple custom CA certificates in to the trust store.
func TestMultipleCustomCAs(t *testing.T) {
	os.Setenv(envDisableSystemRoots, "true")
	os.Setenv(envCaCerts, caRootCertFileLocation+","+caIntCertFileLocation)
	testTLSConfig := createTestTLSConfig(t)

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
	os.Unsetenv(envDisableSystemRoots)
	os.Unsetenv(envCaCerts)
	testTLSConfig := createTestTLSConfig(t)

	// its possible that the system does not have a trust store (minimal docker container for example)
	if len(testTLSConfig.RootCAs.Subjects()) < 1 {
		t.Errorf("after loading system trust store we still have 0. Do you have a system trust store?")
	}
}

// TestSystemRootCAsAndCustomCAs should test that we can load
// both system CAs and custom CAs into trust store
func TestSystemRootCAsAndCustomCAs(t *testing.T) {
	os.Unsetenv(envDisableSystemRoots)
	os.Unsetenv(envCaCerts)
	testTLSConfig := createTestTLSConfig(t)
	systemCACount := len(testTLSConfig.RootCAs.Subjects())

	os.Setenv(envCaCerts, caRootCertFileLocation)
	testTLSConfig = createTestTLSConfig(t)
	currentCACount := len(testTLSConfig.RootCAs.Subjects())

	if currentCACount != (systemCACount + 1) {
		t.Errorf("expected %d certs in trust store but got %d", systemCACount+1, currentCACount)
	}
}

// TestLoadingClientCertAndKey: should test the behaviour of loading
// a pem encoded client x509 certificate and private key
func TestLoadingClientCertAndKey(t *testing.T) {
	os.Unsetenv(envDisableSystemRoots)
	os.Unsetenv(envCaCerts)
	os.Setenv(envClientCert, clientCertFileLocation)
	os.Setenv(envClientKey, clientKeyFileLocation)
	testTLSConfig := createTestTLSConfig(t)

	if len(testTLSConfig.Certificates) < 1 {
		t.Error("failed to load client certficate and key")
	}
}

// TestTLSHardening should test the behaviour of enabling TLS hardening
func TestTLSHardening(t *testing.T) {
	os.Unsetenv(envDisableSystemRoots)
	os.Unsetenv(envCaCerts)
	os.Unsetenv(envClientCert)
	os.Unsetenv(envClientKey)
	os.Setenv(envTLSHardening, "true")
	testTLSConfig := createTestTLSConfig(t)

	if testTLSConfig.MinVersion != hardenedMinVersion {
		t.Error("MinVersion is not set to expected value")
	}
	if testTLSConfig.InsecureSkipVerify {
		t.Error("InsecureSkipVerify is set to true when it should be false")
	}
	if len(testTLSConfig.CipherSuites) == 0 {
		t.Error("CipherSuites is not set")
	}
	if len(testTLSConfig.CurvePreferences) == 0 {
		t.Error("CurvePreferences is not set")
	}
	for i := range testTLSConfig.CipherSuites {
		if testTLSConfig.CipherSuites[i] != hardenedCiphers[i] {
			t.Error("discrepency found in CipherSuites")
		}
	}
	for i := range testTLSConfig.CurvePreferences {
		if testTLSConfig.CurvePreferences[i] != hardenedCurvePreferences[i] {
			t.Error("discrepency found in CurvePreferences")
		}
	}
}
