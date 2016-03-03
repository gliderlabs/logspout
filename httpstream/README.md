# httpstream

You can use these chunked transfer streaming endpoints for quick debugging with `curl` or for setting up easy TCP subscriptions to log sources. They also support WebSocket upgrades.

	GET /logs
	GET /logs/id:<container-id>
	GET /logs/name:<container-name-pattern>

You can select specific log types from a source using a comma-delimited list in the query param `source`. Right now the only sources are `stdout` and `stderr`.

If you include a request `Accept: application/json` header, the output will be JSON objects. Note that when upgrading to WebSocket, it will always use JSON.

Since `/logs` and `/logs/name:<string>` endpoints can return logs from multiple containers, they will by default return color-coded loglines prefixed with the name of the container. You can turn off the color escape codes with query param `colors=off` or the alternative is to stream the data in JSON format, which won't use colors or prefixes.

