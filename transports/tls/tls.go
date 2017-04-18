package tls

import (
	"crypto/tls"
	"crypto/x509"
	"io/ioutil"
	"log"
	"net"
	"os"
	"path/filepath"

	"github.com/gliderlabs/logspout/adapters/raw"
	"github.com/gliderlabs/logspout/router"
)

// default CA and certificate path
const (
	CAPath   = "/mnt/ca/"
	CertPath = "/mnt/cert/"
)

func init() {
	router.AdapterTransports.Register(new(tlsTransport), "tls")
	// convenience adapters around raw adapter
	router.AdapterFactories.Register(rawTLSAdapter, "tls")
}

func getopt(name, dfault string) string {
	value := os.Getenv(name)
	if value == "" {
		value = dfault
	}
	return value
}

func rawTLSAdapter(route *router.Route) (router.LogAdapter, error) {
	route.Adapter = "raw+tls"
	return raw.NewRawAdapter(route)
}

func getCAs(path string) *x509.CertPool {
	cafiles, err := ioutil.ReadDir(path)
	if err != nil {
		return nil
	}
	capool := x509.NewCertPool()
	// TODO: don't khow how to check pool is empty
	empty := true
	for _, f := range cafiles {
		if !f.IsDir() {
			cacert, err := ioutil.ReadFile(path + f.Name())
			if err != nil {
				log.Printf("Can't read CA certificate %v: %v", path+f.Name(), err)
				continue
			}
			ok := capool.AppendCertsFromPEM(cacert)
			if !ok {
				log.Printf("Bad CA certificate %v", path+f.Name())
			}
			empty = !ok && empty
		}
	}

	if empty {
		capool = nil
	}

	return capool
}

func getCertificates(path string) []tls.Certificate {
	var certs []tls.Certificate
	certfiles, err := ioutil.ReadDir(path)
	if err != nil {
		return certs
	}
	for _, f := range certfiles {
		fname := f.Name()

		fext := filepath.Ext(fname)

		fname = fname[0 : len(fname)-len(fext)]

		if fext != ".crt" && fext != ".cert" {
			continue
		}

		keyfile := path + fname + ".key"
		if _, err := os.Stat(keyfile); err != nil {
			continue
		}

		cert, err := tls.LoadX509KeyPair(path+fname+fext, keyfile)
		if err != nil {
			continue
		}

		certs = append(certs, cert)
	}

	return certs
}

type tlsTransport int

func (t *tlsTransport) Dial(addr string, options map[string]string) (net.Conn, error) {

	capath := getopt("CA_PATH", CAPath)
	certpath := getopt("CERT_PATH", CertPath)

	capool := getCAs(capath)

	certs := getCertificates(certpath)

	config := tls.Config{Certificates: certs, RootCAs: capool}

	conn, err := tls.Dial("tcp", addr, &config)

	if err != nil {
		return nil, err
	}
	return conn, nil
}
