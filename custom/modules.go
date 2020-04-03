package main

import (
	_ "github.com/looplab/logspout-logstash"

	_ "github.com/gliderlabs/logspout/adapters/syslog"
	_ "github.com/gliderlabs/logspout/transports/tcp"
	_ "github.com/gliderlabs/logspout/transports/tls"
	_ "github.com/gliderlabs/logspout/transports/udp"
)
