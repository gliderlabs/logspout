NAME=logspout
VERSION=$(shell cat VERSION)

dev:
	docker build -f Dockerfile.dev -t $(NAME):dev .
	docker run --rm \
		-v /var/run/docker.sock:/tmp/docker.sock \
		-p 8000:8000 \
		$(NAME):dev

build:
	mkdir -p build
	docker build -t $(NAME):$(VERSION) .
	docker save $(NAME):$(VERSION) | gzip -9 > build/$(NAME)_$(VERSION).tgz

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

.PHONY: release
