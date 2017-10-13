package main

import (
	_ "github.com/deliveroo/logspout/adapters/syslog"
	_ "github.com/deliveroo/logspout/transports/tcp"
	_ "github.com/deliveroo/logspout/transports/tls"
	_ "github.com/deliveroo/logspout/transports/udp"
	_ "github.com/looplab/logspout-logstash"
)
