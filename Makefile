PWD := $(shell pwd)

all:
	docker build -t logspout-build .
	docker run --rm -v ${PWD}/image:/image:rw logspout-build cp /gopath/bin/logspout /image/
	docker build -t logspout image

release:
	docker tag logspout progrium/logspout
	docker push progrium/logspout

.PHONY: clean
clean:
	rm -rf image/logspout
