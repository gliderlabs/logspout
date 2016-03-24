
FROM gliderlabs/alpine:3.1
ENTRYPOINT ["/bin/logspout"]
VOLUME /mnt/routes
EXPOSE 80

RUN echo http://dl-4.alpinelinux.org/alpine/edge/community >> /etc/apk/repositories \
        && apk add --update go git mercurial

ENV GOPATH=/go
RUN git config --global http.sslVerify false
RUN go get github.com/fsouza/go-dockerclient
RUN (go get github.com/golang/net ||true) && mkdir -p /go/src/golang.org/x && mv /go/src/github.com/golang/net /go/src/golang.org/x/
RUN go get github.com/gorilla/mux
RUN go get github.com/looplab/logspout-logstash

RUN mkdir -p /go/src/github.com/gliderlabs/logspout

COPY .  /go/src/github.com/gliderlabs/logspout

RUN cd /go/src/github.com/gliderlabs/logspout && go build -ldflags "-X main.Version=$(cat VERSION)" -o /bin/logspout

RUN apk del go git mercurial \
        && rm -rf /go \
        && rm -rf /var/cache/apk/*

