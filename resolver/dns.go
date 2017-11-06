package resolver

import (
	"strings"

	"github.com/benschw/srv-lb/lb"
)

// DNSConfig accepts an address and optional load balancer config
type DNSConfig struct {
	Addr  string
	LbCfg *lb.Config
}

// ResolveSrvAddr returns a load-balanced host:port based on DNS SRV lookup (or `addr` if not a hostname)
func ResolveSrvAddr(dnsCfg DNSConfig) (string, error) {
	addr := dnsCfg.Addr

	if v := strings.Split(addr, ":"); len(v) < 2 {
		var cfg *lb.Config
		var err error
		if dnsCfg.LbCfg == nil {
			cfg, err = lb.DefaultConfig()

			if err != nil {
				return addr, err
			}
		} else {
			cfg = dnsCfg.LbCfg
		}

		l := lb.New(cfg, addr)
		resolvAddr, err := l.Next()

		if err == nil {
			addr = resolvAddr.String()
		}
	}

	return addr, nil
}
