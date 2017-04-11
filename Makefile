.PHONY: build

NAME=logspout
VERSION=$(shell cat VERSION)
ifeq ($(shell uname), Darwin)
	XARGS_ARG="-L1"
endif
GOLINT := go list ./... | egrep -v '/custom/|/vendor/' | xargs $(XARGS_ARG) golint | egrep -v 'extpoints.go|types.go'
# max image size of 40MB
MAX_IMAGE_SIZE := 40000000

build-dev:
	docker build -f Dockerfile.dev -t $(NAME):dev .

dev: build-dev
	@docker run --rm \
		-e DEBUG=true \
		-v /var/run/docker.sock:/var/run/docker.sock \
		-v $(PWD):/go/src/github.com/gliderlabs/logspout \
		-p 8000:80 \
		-e ROUTE_URIS=$(ROUTE) \
		$(NAME):dev

build:
	mkdir -p build
	docker build -t $(NAME):$(VERSION) .
	docker save $(NAME):$(VERSION) | gzip -9 > build/$(NAME)_$(VERSION).tgz

lint:
	test -x $(GOPATH)/bin/golint || go get github.com/golang/lint/golint
	go get \
		&& go install \
		&& ls -d */ | egrep -v 'custom/|vendor/' | xargs $(XARGS_ARG) go tool vet -v
	@if [ -n "$(shell $(GOLINT) | cut -d ':' -f 1)" ]; then $(GOLINT) && exit 1 ; fi

test: build-dev
	docker run \
		-v /var/run/docker.sock:/var/run/docker.sock \
		-v $(PWD):/go/src/github.com/gliderlabs/logspout \
		$(NAME):dev go test -v ./router/...

test-image-size:
	@if [ $(shell docker inspect -f '{{ .Size }}' $(NAME):$(VERSION)) -gt $(MAX_IMAGE_SIZE) ]; then \
		echo ERROR: image size greater than $(MAX_IMAGE_SIZE); \
		exit 2; \
	fi

test-build-custom:
	docker tag $(NAME):$(VERSION) gliderlabs/$(NAME):master
	cd custom && docker build -t $(NAME):custom .
	docker run --name $(NAME)-custom $(NAME):custom || true
	docker logs $(NAME)-custom | grep -q logstash
	docker rmi gliderlabs/$(NAME):master || true
	docker rm $(NAME)-custom || true

release:
	rm -rf release && mkdir release
	go get github.com/progrium/gh-release/...
	cp build/* release
	gh-release create gliderlabs/$(NAME) $(VERSION) \
		$(shell git rev-parse --abbrev-ref HEAD) $(VERSION)

circleci:
	rm ~/.gitconfig
ifneq ($(CIRCLE_BRANCH), release)
	echo build-$$CIRCLE_BUILD_NUM > VERSION
endif

clean:
	rm -rf build/
	docker rm $(shell docker ps -aq) || true
	docker rmi $(NAME):dev $(NAME):$(VERSION) || true
	docker rmi $(shell docker images -f 'dangling=true' -q) || true

.PHONY: release clean
