FROM gliderlabs/alpine:3.3
ENTRYPOINT ["/bin/logspout"]
VOLUME /mnt/routes
EXPOSE 80

COPY . /src
RUN cd /src && ./build.sh "$(cat VERSION)"

ONBUILD COPY . /src/
ONBUILD RUN cd /src && ./build.sh "$(cat VERSION)-custom"
