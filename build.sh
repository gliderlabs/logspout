#!/bin/sh
set -e
apk add --update go git mercurial

mkdir -p /go/src

export GOPATH=/go
cd /go/src && wget http://www.golangtc.com/static/download/packages/code.google.com.p.go.net.tar.gz
cd /go/src && tar zxvf code.google.com.p.go.net.tar.gz && rm code.google.com.p.go.net.tar.gz && go install code.google.com/p/go.net/websocket

mv /src/ssl /root
rm /etc/localtime && cd /src && mv localtime /etc
mkdir -p /go/src/github.com/gliderlabs
cp -r /src /go/src/github.com/gliderlabs/logspout
cd /go/src/github.com/gliderlabs/logspout
#go get
go get github.com/Dataman-cloud/logspout/utils
go get github.com/fsouza/go-dockerclient
go get github.com/gorilla/mux
go get github.com/joeshaw/iso8601
go build -ldflags "-X main.Version $1" -o /bin/logspout
apk del go git mercurial
rm -rf /go
rm -rf /var/cache/apk/*

# backwards compatibility
ln -fs /tmp/docker.sock /var/run/docker.sock
