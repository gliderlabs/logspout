package main

import (
	_ "github.com/looplab/logspout-logstash"

	_ "github.com/gliderlabs/logspout/v3/adapters/syslog"
	_ "github.com/gliderlabs/logspout/v3/transports/tcp"
	_ "github.com/gliderlabs/logspout/v3/transports/tls"
	_ "github.com/gliderlabs/logspout/v3/transports/udp"
)
