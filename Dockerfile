FROM gliderlabs/alpine:3.6
ENTRYPOINT ["/src/entrypoint.sh"]
VOLUME /mnt/routes
EXPOSE 80

COPY ./build-env.sh /src/
RUN cd /src && ./build-env.sh
COPY ./build-glide.sh ./glide.yaml ./glide.lock /src/
RUN cd /src && ./build-glide.sh
COPY . /src/
RUN chmod +x /src/entrypoint.sh
RUN apk update
RUN apk add curl
RUN cd /src && ./build.sh "$(cat VERSION)"
