package main

import (
	_ "github.com/ruguoapp/logspout/adapters/gelf"
	_ "github.com/ruguoapp/logspout/adapters/raw"
	_ "github.com/ruguoapp/logspout/adapters/syslog"
	_ "github.com/ruguoapp/logspout/httpstream"
	_ "github.com/ruguoapp/logspout/routesapi"
	_ "github.com/ruguoapp/logspout/transports/tcp"
	_ "github.com/ruguoapp/logspout/transports/tls"
	_ "github.com/ruguoapp/logspout/transports/udp"
)
