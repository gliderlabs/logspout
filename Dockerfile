FROM gliderlabs/alpine:3.6
ENTRYPOINT ["/bin/logspout"]
VOLUME /mnt/routes
EXPOSE 80

COPY ./build-env.sh /src/
RUN cd /src && ./build-env.sh
COPY . /src/
RUN cd /src && ./build.sh "$(cat VERSION)"

ONBUILD COPY ./build-env.sh /src/
ONBUILD RUN cd /src && ./build-env.sh
ONBUILD COPY ./modules.go /src/modules.go
ONBUILD RUN cd /src && ./build.sh "$(cat VERSION)-custom"
