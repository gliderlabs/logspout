# HTTP Adapter


### Usage

The route URI for an HTTP/HTTPS point endpoint should just include the hostname. The HTTP path currently has to specified as a parameter. For example, for Sumo Logic the endpoint URI with for an HTTP collector endpoint would look like this:

```
https://collectors.sumologic.com/receiver/v1/http/SUMO_HTTP_TOKEN
```

But for Logspout, it needs to be written like this:

```
https://collectors.sumologic.com?http.path=/receiver/v1/http/SUMO_HTTP_TOKEN
```

The HTTP adapter also supports 2 parameters to control the buffer capacity and timeout used to determine when the buffer is being flushed if the capacity of the buffer isn't reached in time. The default values are 100 for the buffer capacity and 1000ms for the timeout. The parameters are specified in the URI. For example, to change the timeout to 30 seconds, and make a buffer of only 5 messages, use this URI.

```
https://collectors.sumologic.com?http.path=/receiver/v1/http/SUMO_HTTP_TOKEN\&http.buffer.timeout=30s\&http.buffer.capacity=5
```


### Development 

This assumes that the unique token for the Sumo Logic HTTP collector endpoint is in the environment as ```$SUMO_HTTP_TOKEN```.

```bash
$ DEBUG=1 \ 
   ROUTE=https://collectors.sumologic.com?http.buffer.timeout=30s\&http.buffer.capacity=5\&http.path=/receiver/v1/http/$SUMO_HTTP_TOKEN \
   make dev
```

To create some test messages

```bash
$ docker run --rm -i --name test1 ubuntu bash -c 'NEW_UUID=$(cat /dev/urandom | tr -dc 'a-zA-Z0-9' | fold -w 32 | head -n 1); for i in `seq 1 10`; do echo $NEW_UUID Hello $i; sleep 1; done'
```


### Todos

- [ ] Deal with errors and non-200 responses... somehow
- [ ] Make sure we send back the AWS ELB cookie if we get one
- [ ] Add compression option



### Issues Found While Writing This Adapter

Log of stuff I ran into to verify, validate, discuss, fix, ...

* Cannot figure out how to create a "custom" build from my own source code
* Full URL is not passed as part of Route, so Sumo-style endpoint URL where the path is relevant and includes auth info isn't working
* Uninitialized options map when using non-standard parameter (#75, #76)
* Logspout seems to need ```-e LOGSPOUT=ignore``` to prevent getting into a feedback loop when using debug output - need to validate this feedback loop is a universal possibility or just something i backed myself into
* Makefile needs quotes around ```$(ROUTE)``` if URL includes ampersand, and ampersand needs to be quoted
```bash
$ DEBUG=1 \
    ROUTE=https://collectors.sumologic.com?http.buffer.timeout=30s\& make dev
```
* Docker 1.6 with ```--log-driver=none``` or ```--log-driver=syslog``` will break Logspout



### Issue With --log-driver In Docker 1.6

Just writing this down here for now so I don't lose it... Likely should be discussed in the Docker context, not the Logspout context. Logspout will not see any container output if the ```--log-driver``` (new in Docker 1.6) is set to anything but the default (```json-file```). Proof:

```bash
$ docker run --rm -it --name test1 ubuntu bash -c 'NEW_UUID=$(cat /dev/urandom | tr -dc 'a-zA-Z0-9' | fold -w 32 | head -n 1); for i in `seq 1 10000`; do echo $NEW_UUID Hello $i; sleep 1; done'
$ CID=$(docker ps -l -q)
$ echo -e "GET /containers/c4074eb48952/logs?stdout=1 HTTP/1.0\r\n" | nc -U /var/run/docker.sock
```

This should return the the logs of the started container.

```bash
$ docker stop $CID
$ docker run --rm -it --log-driver=syslog --name test1 ubuntu bash -c 'NEW_UUID=$(cat /dev/urandom | tr -dc 'a-zA-Z0-9' | fold -w 32 | head -n 1); for i in `seq 1 10000`; do echo $NEW_UUID Hello $i; sleep 1; done'
$ CID=$(docker ps -l -q)
$ echo -e "GET /containers/c4074eb48952/logs?stdout=1 HTTP/1.0\r\n" | nc -U /var/run/docker.sock
```

Will return:

```
"logs" endpoint is supported only for "json-file" logging driver
Error running logs job: "logs" endpoint is supported only for "json-file" logging driver
```

This is unfortunate because it prevents using --log-driver=none in conjunction with Logspout to forward logs without touching the host disk. With ```json-file``` being required, the issue of logs running the host of disk space still remains.


### Using The Image From Docker Hub

This again assumes that the unique token for the Sumo Logic HTTP collector endpoint is in the environment as ```$SUMO_HTTP_TOKEN```.

```bash
$ docker run -e DEBUG=1 \
    -v=/var/run/docker.sock:/var/run/docker.sock \
	raychaser/logspout:latest-http-buffered \   
	https://collectors.sumologic.com?http.buffer.timeout=1s\&http.buffer.capacity=100\&http.path=/receiver/v1/http/$SUMO_HTTP_TOKEN
```


