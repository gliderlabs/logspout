FROM flynn/busybox
MAINTAINER Jeff Lindsay <progrium@gmail.com>

ADD ./stage/logspout /bin/logspout
ADD run.sh /home/
WORKDIR /home
RUN chmod +x run.sh

ENV DOCKER unix:///tmp/docker.sock
ENV ROUTESPATH /mnt/routes
VOLUME /mnt/routes

EXPOSE 8000

#ENTRYPOINT ["/home/run.sh"]
CMD []
