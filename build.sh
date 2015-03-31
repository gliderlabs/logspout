#!/bin/sh
set -e
apk add --update go git mercurial
mkdir -p /go/src/github.com/gliderlabs
cp -r /src /go/src/github.com/gliderlabs/logspout
cd /go/src/github.com/gliderlabs/logspout
export GOPATH=/go
go get
go build -ldflags "-X main.Version $1" -o /bin/logspout
rm -rf /go
apk del go git mercurial
rm -rf /var/cache/apk/*
