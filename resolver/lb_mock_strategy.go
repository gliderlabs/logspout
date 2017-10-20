package resolver

import (
	"github.com/benschw/srv-lb/dns"
	"github.com/benschw/srv-lb/lb"
)

// MockStrategy is used for testing DNS load balancing
const MockStrategy lb.StrategyType = "mock"

// New creates a new instance of the load balancer
func New(lib dns.Lookup) lb.GenericLoadBalancer {
	lb := new(MockClb)
	lb.dnsLib = lib
	return lb
}

// MockClb contains the dnslib
type MockClb struct {
	dnsLib dns.Lookup
}

// Next gets the next server in the available nodes
func (lb *MockClb) Next(name string) (dns.Address, error) {
	return dns.Address{Address: "1.2.3.4", Port: 1234}, nil
}

func init() {
	lb.RegisterStrategy(MockStrategy, New)
}
