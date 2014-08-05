FROM flynn/busybox
MAINTAINER CMGS <ilskdw@gmail.com>

ADD ./logspout /bin/logspout

EXPOSE 8000

ENTRYPOINT ["/bin/logspout"]
CMD []
