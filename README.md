# lenz

Fork from logspout by progrium, modified for sending JSON-formatted data to backends. It removed http api interface and changed route file syntax.

I made lenz support multiple mixed protocols backend. When events coming, it will choice one and send the event. Here I use consistent hash for scaling and failover.

In the end, I implement the reload route files method by HUP signal. It would help you dynamic forward events easily.

# logspout

A log router for Docker container output that runs entirely inside Docker. It attaches to all containers on a host, then routes their logs to wherever you want.

It's a 100% stateless log appliance (unless you persist routes). It's not meant for managing log files or looking at history. It is just a means to get your logs out to live somewhere else, where they belong.

For now it only captures stdout and stderr, but soon Docker will let us hook into more ... perhaps getting everything from every container's /dev/log. 

#### Route all container output to remote backends

The simplest way to use lenz is to just take all logs and ship to remotes. Just pass default target URIs as the command.

	$ ./lenz -forwards=udp://zzzz:50433,udp://yyy:50433

Logs will be tagged with the container name. And the appname will be tagged with the first world of the container name.

### Routes Resource

Routes let you configure lenz to hand-off logs to another system.

#### Creating a route

Saving a JSON object in a file like this:

	{
		"source": {
			"filter": "test"
			"types": ["stdout"]
		},
		"target": {
			"addr": [
                "udp://logstash1:50433",
                "udp://logstash2:50433",
            ],
			"append_tag": ".test"
		}
	}

The `source` field should be an object with `filter`, `name`, or `id` fields. You can specify specific log types with the `types` field to collect only `stdout` or `stderr`. If you don't specify `types`, it will route all types. If you specified `filter`, it would filter events by container name. 

To route all logs of all types on all containers, don't specify a `filter`. 

The `append_tag` field of `target` is optional and specific to `logstash`. It lets you append to the tag of events for this route. By default the tag is empty, so an `append_tag` value of `test` would make the tag `test`.

And yes, you can just specify an IP and port for `addr`, but you can also specify a name that resolves via DNS to one or more SRV records.

## License

BSD
