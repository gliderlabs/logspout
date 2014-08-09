build/container: build/lenz Dockerfile
	docker build --no-cache -t lenz .
	touch build/container

build/lenz: *.go
	go build -o build/lenz

release:
	docker tag lenz CMGS/lenz
	docker push CMGS/lenz

.PHONY: clean
clean:
	rm -rf build
