package main

import (
	_ "github.com/deliveroo/logspout/adapters/raw"
	_ "github.com/deliveroo/logspout/adapters/syslog"
	_ "github.com/deliveroo/logspout/httpstream"
	_ "github.com/deliveroo/logspout/routesapi"
	_ "github.com/deliveroo/logspout/transports/tcp"
	_ "github.com/deliveroo/logspout/transports/udp"
	_ "github.com/deliveroo/logspout/transports/tls"
)
