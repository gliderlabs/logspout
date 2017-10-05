FROM gliderlabs/alpine:3.6
ENTRYPOINT ["/bin/logspout"]
VOLUME /mnt/routes
EXPOSE 80

COPY ./build-env.sh /src/
RUN cd /src && ./build-env.sh
COPY ./build-glide.sh ./glide.yaml ./glide.lock modules.go /src/
RUN cd /src && ./build-glide.sh
COPY . /src/
RUN cd /src && ./build.sh "$(cat VERSION)"
