FROM iron/go:dev
ENTRYPOINT ["/bin/logspout"]
VOLUME /mnt/routes
EXPOSE 80

ENV GOPATH=/go
ENV LSPATH=/go/src/github.com/gliderlabs/logspout
RUN mkdir -p $LSPATH

ADD . $LSPATH
RUN cd $LSPATH && go build -o /bin/logspout

ONBUILD COPY ./modules.go ${LSPATH}/modules.go
ONBUILD RUN cd $LSPATH && go get
ONBUILD RUN cd $LSPATH && go build -o /bin/logspout
