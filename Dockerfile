FROM flynn/busybox
MAINTAINER CMGS <ilskdw@gmail.com>

ADD ./lenz /bin/lenz

ENTRYPOINT ["/bin/lenz"]
CMD []
