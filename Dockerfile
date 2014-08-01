# This Dockerfile is used to build the logspout binary only. See
# image/Dockerfile for building a much leaner release image.

FROM google/golang
ADD . /gopath/src/logspout
RUN go get -v logspout
