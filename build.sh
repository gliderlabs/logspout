#!/bin/sh
set -e

rsync -ar /src/ /go/src/github.com/gliderlabs/logspout --exclude glide.lock --exclude glide.yaml

cd /go/src/github.com/gliderlabs/logspout
#ls -lah
export GOPATH=/go
$GOPATH/bin/glide update && $GOPATH/bin/glide install
go build -ldflags "-X main.Version=$1" -o /bin/logspout
apk del go git mercurial build-base
rm -rf /go /var/cache/apk/* /root/.glide

# backwards compatibility
ln -fs /tmp/docker.sock /var/run/docker.sock
