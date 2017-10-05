#!/bin/sh
set -e
mkdir -p /go/src/github.com/gliderlabs
cp -r /src /go/src/github.com/gliderlabs/logspout
cd /go/src/github.com/gliderlabs/logspout
export GOPATH=/go
go get github.com/Masterminds/glide && $GOPATH/bin/glide update && $GOPATH/bin/glide install
