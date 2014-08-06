FROM flynn/busybox
MAINTAINER CMGS <ilskdw@gmail.com>

ADD ./logspout /bin/lenz

ENTRYPOINT ["/bin/lenz"]
CMD []
