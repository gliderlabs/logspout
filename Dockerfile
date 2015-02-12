FROM gliderlabs/alpine:3.1
ENTRYPOINT ["/bin/logspout"]
VOLUME /mnt/routes
EXPOSE 8000

COPY . /go/src/github.com/progrium/logspout
RUN apk-install go git mercurial \
	&& cd /go/src/github.com/progrium/logspout \
	&& export GOPATH=/go \
	&& go get \
	&& go build -o /bin/logspout \
	&& rm -rf /go \
	&& apk del go git mercurial
