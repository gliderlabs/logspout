# Custom Logspout Builds

Forking logspout to change modules is unnecessary! Instead, you can create an
empty Dockerfile based on `gliderlabs/logspout:master` and include a new
`modules.go` file as well as the `build.sh` script that resides in the root of
this repo for the build context that will override the standard one.

This directory is an example of doing this. It pairs logspout down to just the
syslog adapter and TCP transport. Note this means you can only create routes
with `syslog+tcp` as the adapter.

It also shows you can take this opportunity to change default configuration by
setting environment in the Dockefile. Here we change the syslog adapter format
from the default of `rfc5424` to old school `rfc3164`.

Now you just have to `docker build` with this Dockerfile and you'll get a custom
logspout container image. No need to install Go, no need to maintain a fork.
