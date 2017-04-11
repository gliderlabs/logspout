FROM golang:1.8-alpine
ENTRYPOINT ["/bin/logspout"]
VOLUME /mnt/routes
EXPOSE 80

ENV GOPATH=/go
ENV LSPATH=/go/src/github.com/gliderlabs/logspout
RUN apk update && apk add --update git

RUN mkdir -p $LSPATH
ADD . $LSPATH
RUN cd $LSPATH && go get github.com/Masterminds/glide && $GOPATH/bin/glide install
RUN cd $LSPATH && go build -o /bin/logspout
RUN apk del git pcre expat libcurl libssh2
RUN rm -rf $GOPATH/bin/glide $GOPATH/src/github.com/Masterminds/glide /root/.glide /tmp/* /root/* /var/cache/apk/*

ONBUILD COPY ./modules.go ${LSPATH}/modules.go
ONBUILD RUN cd $LSPATH && go get
ONBUILD RUN cd $LSPATH && go build -o /bin/logspout
