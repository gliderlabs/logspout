Instructions on how to build/test your own modules. 

## Getting Started

1. Fork this repository
1. Create a new repository for your adapter
1. Copy something like [raw.go](https://github.com/gliderlabs/logspout/blob/master/adapters/raw/raw.go) to get started.
1. Add your module to modules.go

> You'll need to add the `build.sh` from this repository to the directory from which you run `docker build` or you will get errors

Now build and run logspout with your adapter, replace SYSLOG with your own syslog url. 

```sh
SYSLOG=syslog://logs.papertrailapp.com:55555 ./run-custom.sh
```

Now let's add your new adapter to the running logspout (replace address below with your final stats destination):

```sh
curl http://localhost:8000/routes -d '{
  "adapter": "myadapter",
  "filter_sources": ["stdout" ,"stderr"],
  "address": "localhost:1234"
}'
```

Now any log messages that come out of any container on your machine will go through your adapter. 
