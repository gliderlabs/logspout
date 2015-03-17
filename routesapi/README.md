# routesapi

### Routes Resource

Routes let you configure logspout to hand-off logs to another system using logspout adapters, such as syslog.

#### Creating a route

	POST /routes

Takes a JSON object like this:

	{
		"source": {
			"filter": "_db"
			"types": ["stdout"]
		},
		"target": {
			"type": "syslog",
			"addr": "logaggregator.service.consul"
			"append_tag": ".db"
		}
	}

The `source` field should be an object with `filter`, `name`, `prefix`, or `id` fields. `prefix` allows a string match against the start of a container name (e.g. "frontend" will match containers named like "frontend-1"). You can specify specific log types with the `types` field to collect only `stdout` or `stderr`. If you don't specify `types`, it will route all types.

To route all logs of all types on all containers, don't specify a `source`.

The `append_tag` field of `target` is optional and specific to `syslog`. It lets you append to the tag of syslog packets for this route. By default the tag is `<container-name>`, so an `append_tag` value of `.app` would make the tag `<container-name>.app`.

And yes, you can just specify an IP and port for `addr`, but you can also specify a name that resolves via DNS to one or more SRV records. That means this works great with [Consul](http://www.consul.io/) for service discovery.

#### Listing routes

	GET /routes

Returns a JSON list of current routes:

	[
		{
			"id": "3631c027fb1b",
			"source": {
				"name": "mycontainer"
			},
			"target": {
				"type": "syslog",
				"addr": "192.168.1.111:514"
			}
		}
	]

#### Viewing a route

	GET /routes/<id>

Returns a JSON route object:

	{
		"id": "3631c027fb1b",
		"source": {
			"id": "a9efd0aeb470"
			"types": ["stderr"]
		},
		"target": {
			"type": "syslog",
			"addr": "192.168.1.111:514"
		}
	}

#### Deleting a route

	DELETE /routes/<id>
