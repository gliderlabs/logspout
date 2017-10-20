package resolver

import (
	"testing"

	"github.com/benschw/srv-lb/lb"
)

func TestSrvLookup(t *testing.T) {
	lbCfg, _ := lb.DefaultConfig()
	lbCfg.Strategy = MockStrategy
	dnsCfg := DNSConfig{Addr: "foo.example.com", LbCfg: lbCfg}

	addr, _ := ResolveSrvAddr(dnsCfg)

	if addr != "1.2.3.4:1234" {
		t.Error("expected address string of 1.2.3.4:1234, got:", addr)
	}
}

func TestIpPassthrough(t *testing.T) {
	dnsCfg := DNSConfig{Addr: "10.0.0.1:1234"}

	addr, _ := ResolveSrvAddr(dnsCfg)

	if addr != "10.0.0.1:1234" {
		t.Error("expected output to equal input (10.0.0.1:1234), got:", addr)
	}
}
