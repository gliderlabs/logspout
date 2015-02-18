FROM gliderlabs/alpine:3.1
ENTRYPOINT ["/bin/logspout"]
VOLUME /mnt/routes
EXPOSE 8000

COPY . /go/src/github.com/gliderlabs/logspout
RUN apk-install go git mercurial \
	&& cd /go/src/github.com/gliderlabs/logspout \
	&& export GOPATH=/go \
	&& go get \
	&& go build -ldflags "-X main.Version $(cat VERSION)" -o /bin/logspout \
	&& rm -rf /go \
	&& apk del go git mercurial
