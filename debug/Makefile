MODE ?= udp

debug:
	@docker history logspout-debug &> /dev/null \
		|| docker build -f Dockerfile -t logspout-debug .
ifeq ($(MODE), udp)
	@echo udp mode
	@docker run --rm -e LOGSPOUT=ignore --name logspout-debug logspout-debug nc -lvup 9000
else
	@echo tcp mode
	@docker run --rm -e LOGSPOUT=ignore --name logspout-debug logspout-debug nc -lvp 9000
endif
