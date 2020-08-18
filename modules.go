package main

import (
	_ "github.com/gliderlabs/logspout/v3/adapters/multiline"
	_ "github.com/gliderlabs/logspout/v3/adapters/raw"
	_ "github.com/gliderlabs/logspout/v3/adapters/syslog"
	_ "github.com/gliderlabs/logspout/v3/healthcheck"
	_ "github.com/gliderlabs/logspout/v3/httpstream"
	_ "github.com/gliderlabs/logspout/v3/routesapi"
	_ "github.com/gliderlabs/logspout/v3/transports/tcp"
	_ "github.com/gliderlabs/logspout/v3/transports/tls"
	_ "github.com/gliderlabs/logspout/v3/transports/udp"
)
