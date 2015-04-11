# logspout

> Docker Hub automated builds for `gliderlabs/logspout:latest` and `progrium/logspout:latest` are now pointing to the `release` branch. For `master`, use `gliderlabs/logspout:master`. Individual versions are also available as saved images in [releases](https://github.com/gliderlabs/logspout/releases).

Logspout is a log router for Docker containers that runs inside Docker. It attaches to all containers on a host, then routes their logs wherever you want. It also has an extensible module system.

It's a mostly stateless log appliance. It's not meant for managing log files or looking at history. It is just a means to get your logs out to live somewhere else, where they belong.

For now it only captures stdout and stderr, but a module to collect container syslog is planned.

## Getting logspout

Logspout is a very small Docker container (14MB virtual, based on [Alpine](https://github.com/gliderlabs/docker-alpine)). Pull the latest release from the index:

	$ docker pull gliderlabs/logspout:latest

You can also download and load a specific version:

	$ curl -s dl.gliderlabs.com/logspout/v2.tgz | docker load

## Using logspout

#### Route all container output to remote syslog

The simplest way to use logspout is to just take all logs and ship to a remote syslog. Just pass a syslog URI (or several comma separated URIs) as the command. Also, we always mount the Docker Unix socket with `-v` to `/tmp/docker.sock`:

	$ docker run --name="logspout" \
		--volume=/var/run/docker.sock:/tmp/docker.sock \
		gliderlabs/logspout \
		syslog://logs.papertrailapp.com:55555

logspout will gather logs from other containers that are started **without the `-t` option**. 

To see what data is used for syslog messages, see the [syslog adapter](http://github.com/gliderlabs/logspout/blob/master/adapters) docs.

#### Ignoring specific containers

You can tell logspout to ignore specific containers by setting an environment variable when starting your container, like so:-

        $ docker run -d -e 'LOGSPOUT=ignore' image

#### Inspect log streams using curl

Using the [httpstream module](http://github.com/gliderlabs/logspout/blob/master/httpstream), you can connect with curl to see your local aggregated logs in realtime. You can do this without setting up a route URI.

	$ docker run -d --name="logspout" \
		--volume=/var/run/docker.sock:/tmp/docker.sock \
		--publish=127.0.0.1:8000:8000 \
		gliderlabs/logspout
	$ curl http://127.0.0.1:8000/logs

You should see a nicely colored stream of all your container logs. You can filter by container name and more. You can also get JSON objects, or you can upgrade to WebSocket and get JSON logs in your browser.

See [httpstream module](http://github.com/gliderlabs/logspout/blob/master/httpstream) for all options.

#### Create custom routes via HTTP

Using the [routesapi module](http://github.com/gliderlabs/logspout/blob/master/routesapi) logspout can also expose a `/routes` resource to create and manage routes.

	$ curl $(docker port `docker ps -lq` 8000)/routes \
		-X POST \
		-d '{"source": {"filter": "db", "types": ["stderr"]}, "target": {"type": "syslog", "addr": "logs.papertrailapp.com:55555"}}'

That example creates a new syslog route to [Papertrail](https://papertrailapp.com) of only `stderr` for containers with `db` in their name.

Routes are stored on disk, so by default routes are ephemeral. You can mount a volume to `/mnt/routes` to persist them.

See [routesapi module](http://github.com/gliderlabs/logspout/blob/master/routesapi) for all options.

## Modules

The standard distribution of logspout comes with all modules defined in this repository. You can remove or add new modules with custom builds of logspout. Just edit the `modules.go` file and do a `docker build`.

### Builtin modules

 * adapters/raw
 * adapters/syslog
 * conntypes/tcp
 * conntypes/udp
 * httpstream
 * routesapi

### Third-party modules

 * logspout-kafka...
 * logspout-redis...
 * [logspout-logstash](https://github.com/looplab/logspout-logstash)

## Contributing

As usual, pull requests are welcome. You can also propose releases by opening a PR against the `release` branch from `master`. Please be sure to bump the version and update `CHANGELOG.md` and include your changelog text in the PR body.

Discuss logspout development with us on Freenode in `#gliderlabs`.

## Sponsor

This project was made possible by [DigitalOcean](http://digitalocean.com) and [Deis](http://deis.io).

## License

BSD
