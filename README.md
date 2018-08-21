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

> NOTE: Use of this option **may** cause the first few lines of log output to be missed following a container being started, if the container starts outputting logs before logspout has a chance to see them. If consistent capture of *every* line of logs is critical to your application, you might want to test thoroughly and/or avoid this option (at the expense of getting the entire backlog for every restarting container). This does not affect containers that are removed and recreated.


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

#### Multiline logging

In order to enable multiline logging, you must first prefix your adapter with the multiline adapter:

	$ docker run \
		--volume=/var/run/docker.sock:/var/run/docker.sock \
		gliderlabs/logspout \
		multiline+raw://192.168.10.10:5000?filter.name=*_db

Using the the above prefix enables multiline logging on all containers by default. To enable it only to specific containers set MULTILINE_ENABLE_DEFAULT=false for logspout, and use the LOGSPOUT_MULTILINE environment variable on the monitored container:

    $ docker run -d -e 'LOGSPOUT_MULTILINE=true' image

##### MULTILINE_MATCH

Using the environment variable `MULTILINE_MATCH`=<first|last|nonfirst|nonlast> (default `nonfirst`) you define, which lines should be matched to the `MULTILINE_PATTERN`.
* first: match first line only and append following messages until you match another line
* last: concatenate all messages until the pattern matches the next line
* nonlast: match a line, append upcoming matching lines, also append first non-matching line and start
* nonfirst: append all matching lines to first line and start over with the next non-matching line

##### Important!
If you use multiline logging with raw, it's recommended to json encode the Data to avoid line breaks in the output, eg:
    
    "RAW_FORMAT={{ toJSON .Data }}\n"

#### Environment variables

* `ALLOW_TTY` - include logs from containers started with `-t` or `--tty` (i.e. `Allocate a pseudo-TTY`)
* `BACKLOG` - suppress container tail backlog
* `TAIL` - specify the number of lines in the log tail to capture when logspout starts (default `all`)
* `DEBUG` - emit debug logs
* `EXCLUDE_LABEL` - exclude logs with a given label
* `INACTIVITY_TIMEOUT` - detect hang in Docker API (default 0)
* `HTTP_BIND_ADDRESS` - configure which interface address to listen on (default 0.0.0.0)
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
* `MULTILINE_ENABLE_DEFAULT` - enable multiline logging for all containers when using the multiline adapter (default `true`)
* `MULTILINE_MATCH` - determines which lines the pattern should match, one of first|last|nonfirst|nonlast, for details see: [MULTILINE_MATCH](#multiline_match) (default `nonfirst`)
* `MULTILINE_PATTERN` - pattern for multiline logging, see: [MULTILINE_MATCH](#multiline_match) (default: `^\s`)
* `MULTILINE_FLUSH_AFTER` - maximum time between the first and last lines of a multiline log entry in milliseconds (default: 500)
* `MULTILINE_SEPARATOR` - separator between lines for output (default: `\n`)

#### Raw Format

The raw adapter has a function `toJSON` that can be used to format the message/fields to generate JSON-like output in a simple way, or full JSON output.

Use examples:

##### Mixed JSON + generic:
```
{{ .Time.Format "2006-01-02T15:04:05Z07:00" }} { "container" : "{{ .Container.Name }}", "labels": {{ toJSON .Container.Config.Labels }}, "timestamp": "{{ .Time.Format "2006-01-02T15:04:05Z07:00" }}", "source" : "{{ .Source }}", "message": {{ toJSON .Data }} }
```

```
2017-10-26T11:59:32Z { "container" : "/catalogo_worker_1", "image": "sha256:e9bce6c17c80c603c4c8dbac2ad2285982d218f6ea0332f8b0fb84572941b773", "labels": {"com.docker.compose.config-hash":"4f9c3d3bfb2f65e29a4bc8a4a1b3f0a1c8a42323106a5e9106fe9279f8031321","com.docker.compose.container-number":"1","com.docker.compose.oneoff":"False","com.docker.compose.project":"catalogo","com.docker.compose.service":"worker","com.docker.compose.version":"1.16.1","logging":"true"}, "timestamp": "2017-10-26T11:59:32Z", "source" : "stdout", "message": "2017-10-26 11:59:32,950 INFO success: command_bus_0 entered RUNNING state, process has stayed up for \u003e than 1 seconds (startsecs)" }
```

##### Full JSON like:

```
{ "container" : "{{ .Container.Name }}", "labels": {{ toJSON .Container.Config.Labels }}, "timestamp": "{{ .Time.Format "2006-01-02T15:04:05Z07:00" }}", "source" : "{{ .Source }}", "message": {{ toJSON .Data }} }
```

```json
{
  "container": "/a_container",
  "image": "sha256:e9bce6c17c80c603c4c8dbac2ad2285982d218f6ea0332f8b0fb84572941b773",
  "labels": {
    "com.docker.compose.config-hash": "4f9c3d3bfb2f65e29a4bc8a4a1b3f0a1c8a42323106a5e9106fe9279f8031321",
    "com.docker.compose.container-number": "1",
    "com.docker.compose.oneoff": "False",
    "com.docker.compose.project": "a_project",
    "com.docker.compose.service": "worker",
    "com.docker.compose.version": "1.16.1",
    "logging": "true"
  },
  "timestamp": "2017-10-26T11:59:32Z",
  "source": "stdout",
  "message": "2017-10-26 11:59:32,950 INFO success: command_bus_0 entered RUNNING state, process has stayed up for > than 1 seconds (startsecs)"
}

```

#### Using Logspout in a swarm

In a swarm, logspout is best deployed as a global service.  When running logspout with 'docker run', you can change the value of the hostname field using the `SYSLOG_HOSTNAME` environment variable as explained above. However, this does not work in a compose file because the value for `SYSLOG_HOSTNAME` will be the same for all logspout "tasks", regardless of the docker host on which they run. To support this mode of deployment, the syslog adapter will look for the file `/etc/host_hostname` and, if the file exists and it is not empty, will configure the hostname field with the content of this file. You can then use a volume mount to map a file on the docker hosts with the file `/etc/host_hostname` in the container.  The sample compose file below illustrates how this can be done

```
version: "3"
networks:
  logging:
services:
  logspout:
    image: gliderlabs/logspout:latest
    networks:
      - logging
    volumes:
      - /etc/hostname:/etc/host_hostname:ro
      - /var/run/docker.sock:/var/run/docker.sock
    command:
      syslog://svt2-logger.am2.cloudra.local:514
    deploy:
      mode: global
      resources:
        limits:
          cpus: '0.20'
          memory: 256M
        reservations:
          cpus: '0.10'
          memory: 128M
```

logspout can then be deployed as a global service in the swarm with the following command

```bash
docker stack deploy --compose-file <name of your compose file> STACK
```

More information about services and their mode of deployment can be found here:
https://docs.docker.com/engine/swarm/how-swarm-mode-works/services/ 

### TLS Settings
logspout supports modification of the client TLS settings via environment variables described below:

| Environment Variable  | Description |
| :---                  |  :---       |
| `LOGSPOUT_TLS_DISABLE_SYSTEM_ROOTS` | when set to `true` it disables loading the system trust store into the trust store of logspout |
| `LOGSPOUT_TLS_CA_CERTS` | a comma seperated list of filesystem paths to pem encoded CA certificates that should be added to logsput's TLS trust store. Each pem file can contain more than one certificate |
| `LOGSPOUT_TLS_CLIENT_CERT` | filesytem path to pem encoded x509 client certificate to load when TLS mutual authentication is desired |
| `LOGSPOUT_TLS_CLIENT_KEY` | filesytem path to pem encoded client private key to load when TLS mutual authentication is desired |
| `LOGSPOUT_TLS_HARDENING` | when set to `true` it enables stricter client TLS settings designed to mitigate some known TLS vulnerabilities |

#### Example TLS settings
The following settings cover some common use cases.
When running docker, use the `-e` flag to supply environment variables

**add your own CAs to the list of trusted authorities**
```
export LOGSPOUT_TLS_CA_CERTS="/opt/tls/ca/myRootCA1.pem,/opt/tls/ca/myRootCA2.pem"
```

**force logspout to ONLY trust your own CA**
```
export LOGSPOUT_TLS_DISABLE_SYSTEM_ROOTS=true
export LOGSPOUT_TLS_CA_CERTS="/opt/tls/ca/myRootCA1.pem"
```

**configure client authentication**
```
export LOGSPOUT_TLS_CLIENT_CERT="/opt/tls/client/myClient.pem"
export LOGSPOUT_TLS_CLIENT_KEY="/opt/tls/client/myClient-key.pem"
```

**highest possible security settings (paranoid mode)**
```
export LOGSPOUT_TLS_DISABLE_SYSTEM_ROOTS=true
export LOGSPOUT_TLS_HARDENING=true
export LOGSPOUT_TLS_CA_CERTS="/opt/tls/ca/myRootCA1.pem"
export LOGSPOUT_TLS_CLIENT_CERT="/opt/tls/client/myClient.pem"
export LOGSPOUT_TLS_CLIENT_KEY="/opt/tls/client/myClient-key.pem"
```

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

 * [logspout-kafka](https://github.com/dylanmei/logspout-kafka)
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
