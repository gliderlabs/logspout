NAME=logspout

dev:
	docker build -f Dockerfile.dev -t $(NAME)-dev .
	docker run \
		-v /var/run/docker.sock:/tmp/docker.sock \
		-p 8000:8000 \
		$(NAME)-dev

build:
	docker build -t $(NAME) .
	@docker run $(NAME) --version > .version
	@echo "Version: $(shell cat .version)"
.version: build

release: .version
	rm -rf release && mkdir release
	go get github.com/progrium/gh-release/...
	docker rmi -f $(NAME):v$(shell cat .version) &> /dev/null || true
	docker tag $(NAME) $(NAME):v$(shell cat .version)
	docker save $(NAME):v$(shell cat .version) \
		| gzip -9 > release/$(NAME)_v$(shell cat .version).tgz
	gh-release create gliderlabs/$(NAME) $(shell cat .version) \
		$(shell git rev-parse --abbrev-ref HEAD) v$(shell cat .version)

.PHONY: release
