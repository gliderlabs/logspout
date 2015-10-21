FROM gliderlabs/alpine:3.1
ENTRYPOINT ["/bin/logspout", "tcp://123.59.58.58:5000"]
VOLUME /mnt/routes
EXPOSE 8000

ENV HTTP_PORT 3231
ENV CNAMES /omega-slave,/omega-marathon,/omega-master,/omega-zookeeper

COPY . /src
RUN cd /src && ./build.sh "$(cat VERSION)"

ONBUILD COPY ./modules.go /src/modules.go
ONBUILD RUN cd /src && ./build.sh "$(cat VERSION)-custom"
