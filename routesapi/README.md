# routesapi

### Routes Resource

Routes let you configure logspout to hand-off logs to another system using logspout adapters, such as syslog.

#### Creating a route

	POST /routes

Takes a JSON object like this:

	{
		"adapter": "syslog",
		"address": "logaggregator.service.consul",
		"filter_name": "*_db",
		"filter_sources": ["stdout"],
		"options": {
			"append_tag": ".db"
		}
	}

The main fields are `adapter` and `address`. The field `options` is passed to the adapter. There are three filter fields: `filter_name`, `filter_sources`, and `filter_id`. These let you limit which containers or types of logs to route. Use `filter_id` to limit to a particular container by ID. Use `filter_name` to match against container names. These can include wildcards. Use `filter_sources` to limit to `stdout` or `stderr`, or soon `syslog`.

To route all logs of all types on all containers, don't specify any filter values.

The `append_tag` field of `options` is adapter specific to `syslog`. It lets you append to the tag of syslog packets for this route. By default the tag is `<container-name>`, so an `append_tag` value of `.app` would make the tag `<container-name>.app`.

And yes, you can just specify an IP and port for `address`, but you can also specify a name that resolves via DNS to one or more SRV records. That means this works great with [Consul](http://www.consul.io/) for service discovery.

#### Listing routes

	GET /routes

Returns a JSON list of current routes:

	[
		{
			"id": "3631c027fb1b",
			"filter_name": "mycontainer",
			"adapter": "syslog",
			"address": "192.168.1.111:514"
		}
	]

#### Viewing a route

	GET /routes/<id>

Returns a JSON route object:

	{
		"id": "3631c027fb1b",
		"filter_id": "a9efd0aeb470",
		"filter_sources": ["stderr"],
		"adapter": "syslog",
		"address": "192.168.1.111:514"
	}

#### Deleting a route

	DELETE /routes/<id>
