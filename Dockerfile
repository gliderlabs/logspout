FROM alpine:3.6
ENTRYPOINT ["/bin/logspout"]
VOLUME /mnt/routes
EXPOSE 80

COPY . /src
RUN cd /src && ./build.sh "$(cat VERSION)"

ONBUILD COPY ./build.sh /src/build.sh
ONBUILD COPY ./modules.go /src/modules.go
ONBUILD RUN cd /src && chmod +x ./build.sh && ./build.sh "$(cat VERSION)-custom"
