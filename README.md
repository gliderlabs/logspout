# logspout

[![CircleCI](https://img.shields.io/circleci/project/gliderlabs/logspout/release.svg)](https://circleci.com/gh/gliderlabs/logspout)
[![Docker pulls](https://img.shields.io/docker/pulls/gliderlabs/logspout.svg)](https://hub.docker.com/r/gliderlabs/logspout/)
[![IRC Channel](https://img.shields.io/badge/irc-%23gliderlabs-blue.svg)](https://kiwiirc.com/client/irc.freenode.net/#gliderlabs)

> Docker Hub automated builds for `gliderlabs/logspout:latest` and `progrium/logspout:latest` are now pointing to the `release` branch. For `master`, use `gliderlabs/logspout:master`. Individual versions are also available as saved images in [releases](https://github.com/gliderlabs/logspout/releases).

Logspout is a log router for Docker containers that runs inside Docker. It attaches to all containers on a host, then routes their logs wherever you want. It also has an extensible module system.

It's a mostly stateless log appliance. It's not meant for managing log files or looking at history. It is just a means to get your logs out to live somewhere else, where they belong.

For now it only captures stdout and stderr, but a module to collect container syslog is planned.

## Getting logspout

Logspout is a very small Docker container (15.2MB virtual, based on [Alpine](https://github.com/gliderlabs/docker-alpine)). Pull the latest release from the index:

	$ docker pull gliderlabs/logspout:latest

You can also download and load a specific version:

	$ curl -s dl.gliderlabs.com/logspout/v2.tgz | docker load

## Using logspout

#### Route all container output to remote syslog

The simplest way to use logspout is to just take all logs and ship to a remote syslog. Just pass a syslog URI (or several comma separated URIs) as the command. Here we show use of the `tls` encrypted transport option in the URI. Also, we always mount the Docker Unix socket with `-v` to `/var/run/docker.sock`:

	$ docker run --name="logspout" \
		--volume=/var/run/docker.sock:/var/run/docker.sock \
		gliderlabs/logspout \
		syslog+tls://logs.papertrailapp.com:55555

logspout will gather logs from other containers that are started **without the `-t` option** and are configured with a logging driver that works with `docker logs` (`journald` and `json-file`).

To see what data is used for syslog messages, see the [syslog adapter](http://github.com/gliderlabs/logspout/blob/master/adapters) docs.

#### Ignoring specific containers

You can tell logspout to ignore specific containers by setting an environment variable when starting your container, like so:-

    $ docker run -d -e 'LOGSPOUT=ignore' image

Or, by adding a label which you define by setting an environment variable when running logspout:

    $ docker run --name="logspout" \
        -e EXCLUDE_LABEL=logspout.exclude \
        --volume=/var/run/docker.sock:/var/run/docker.sock \
        gliderlabs/logspout
    $ docker run -d --label logspout.exclude=true image

#### Including specific containers

You can tell logspout to only include certain containers by setting filter parameters on the URI:

	$ docker run \
		--volume=/var/run/docker.sock:/var/run/docker.sock \
		gliderlabs/logspout \
		raw://192.168.10.10:5000?filter.name=*_db

	$ docker run \
		--volume=/var/run/docker.sock:/var/run/docker.sock \
		gliderlabs/logspout \
		raw://192.168.10.10:5000?filter.id=3b6ba57db54a

	$ docker run \
		--volume=/var/run/docker.sock:/var/run/docker.sock \
		gliderlabs/logspout \
		raw://192.168.10.10:5000?filter.sources=stdout%2Cstderr

	# Forward logs from containers with both label 'a' starting with 'x', and label 'b' ending in 'y'.
	$ docker run \
		--volume=/var/run/docker.sock:/var/run/docker.sock \
		gliderlabs/logspout \
		raw://192.168.10.10:5000?filter.labels=a:x*%2Cb:*y

Note that you must URL-encode parameter values such as the comma in `filter.sources` and `filter.labels`.

#### Multiple logging destinations

You can route to multiple destinations by comma-separating the URIs:

	$ docker run \
		--volume=/var/run/docker.sock:/var/run/docker.sock \
		gliderlabs/logspout \
		raw://192.168.10.10:5000?filter.name=*_db,syslog+tls://logs.papertrailapp.com:55555?filter.name=*_app

#### Suppressing backlog tail
You can tell logspout to only display log entries since container "start" or "restart" event by setting a `BACKLOG=false` environment variable (equivalent to `docker logs --since=0s`):

	$ docker run -d --name="logspout" \
		-e 'BACKLOG=false' \
		--volume=/var/run/docker.sock:/var/run/docker.sock \
		gliderlabs/logspout

The default behaviour is to output all logs since creation of the container (equivalent to `docker logs --tail=all` or simply `docker logs`).

> NOTE: Use of this option **may** cause the first few lines of log output to be missed following a container being started, if the container starts outputting logs before logspout has a chance to see them. If consistent capture of *every* line of logs is critical to your application, you might want to test thorougly and/or avoid this option (at the expense of getting the entire backlog for every restarting container). This does not affect containers that are removed and recreated.


#### Environment variable, TAIL
Whilst BACKLOG=false restricts the tail by setting the Docker Logs.Options.Since to time.Now(), another mechanism to restrict the tail is to set TAIL=n.  Use of this mechanism avoids parsing the earlier content of the logfile which may have a speed advantage if the tail content is of no interest or has become corrupted.

#### Inspect log streams using curl

Using the [httpstream module](http://github.com/gliderlabs/logspout/blob/master/httpstream), you can connect with curl to see your local aggregated logs in realtime. You can do this without setting up a route URI.

	$ docker run -d --name="logspout" \
		--volume=/var/run/docker.sock:/var/run/docker.sock \
		--publish=127.0.0.1:8000:80 \
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

#### Detecting timeouts in Docker log streams

Logspout relies on the Docker API to retrieve container logs. A failure in the API may cause a log stream to hang. Logspout can detect and restart inactive Docker log streams. Use the environment variable `INACTIVITY_TIMEOUT` to enable this feature. E.g.: `INACTIVITY_TIMEOUT=1m` for a 1-minute threshold.

#### Environment variables

* `ALLOW_TTY` - include logs from containers started with `-t` or `--tty` (i.e. `Allocate a pseudo-TTY`)
* `BACKLOG` - suppress container tail backlog
* `TAIL` - specify the number of lines in the log tail to capture when logspout starts (default `all`)
* `DEBUG` - emit debug logs
* `EXCLUDE_LABEL` - exclude logs with a given label
* `INACTIVITY_TIMEOUT` - detect hang in Docker API (default 0)
* `PORT` or `HTTP_PORT` - configure which port to listen on (default 80)
* `RAW_FORMAT` - log format for the raw adapter (default `{{.Data}}\n`)
* `RETRY_COUNT` - how many times to retry a broken socket (default 10)
* `ROUTESPATH` - path to routes (default `/mnt/routes`)
* `SYSLOG_DATA` - datum for data field (default `{{.Data}}`)
* `SYSLOG_FORMAT` - syslog format to emit, either `rfc3164` or `rfc5424` (default `rfc5424`)
* `SYSLOG_HOSTNAME` - datum for hostname field (default `{{.Container.Config.Hostname}}`)
* `SYSLOG_PID` - datum for pid field (default `{{.Container.State.Pid}}`)
* `SYSLOG_PRIORITY` - datum for priority field (default `{{.Priority}}`)
* `SYSLOG_STRUCTURED_DATA` - datum for structured data field
* `SYSLOG_TAG` - datum for tag field (default `{{.ContainerName}}+route.Options["append_tag"]`)
* `SYSLOG_TIMESTAMP` - datum for timestamp field (default `{{.Timestamp}}`)

## Modules

The standard distribution of logspout comes with all modules defined in this repository. You can remove or add new modules with custom builds of logspout. In the `custom` dir, edit the `modules.go` file and do a `docker build`.

### Builtin modules

 * adapters/raw
 * adapters/syslog
 * transports/tcp
 * transports/tls
 * transports/udp
 * httpstream
 * routesapi

### Third-party modules

 * [logspout-kafka](https://github.com/gettyimages/logspout-kafka)
 * logspout-redis...
 * [logspout-logstash](https://github.com/looplab/logspout-logstash)
 * [logspout-redis-logstash](https://github.com/rtoma/logspout-redis-logstash)
 * [logspout-gelf](https://github.com/micahhausler/logspout-gelf) for Graylog

### Loggly support

Use logspout to stream your docker logs to Loggly via the [Loggly syslog endpoint](https://www.loggly.com/docs/streaming-syslog-without-using-files/).
```
$ docker run --name logspout -d --volume=/var/run/docker.sock:/var/run/docker.sock \
    -e SYSLOG_STRUCTURED_DATA="<Loggly API Key>@41058 tag=\"some tag name\"" \
    gliderlabs/logspout \
    syslog+tcp://logs-01.loggly.com:514
```

## Contributing

As usual, pull requests are welcome. You can also propose releases by opening a PR against the `release` branch from `master`. Please be sure to bump the version and update `CHANGELOG.md` and include your changelog text in the PR body.

Discuss logspout development with us on Freenode in `#gliderlabs`.

## Sponsor

This project was made possible by [DigitalOcean](http://digitalocean.com) and [Deis](http://deis.io).

## License

BSD
<img src="https://ga-beacon.appspot.com/UA-58928488-2/logspout/readme?pixel" />
