FROM gliderlabs/alpine:3.1
ENTRYPOINT ["/bin/logspout"]
VOLUME /mnt/routes
EXPOSE 8000

COPY . /src
RUN cd /src && ./build.sh "$(cat VERSION)"

ONBUILD COPY ./modules.go /src/modules.go
ONBUILD RUN cd /src && ./build.sh "$(cat VERSION)-custom"
