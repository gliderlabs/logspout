set -ex

docker run --rm -v "$GOPATH":/go -w /go/src/github.com/gliderlabs/logspout -e "GOPATH=/go" iron/go:dev sh -c 'go build -o logspout'

# Can build the image too
# docker build -t gliderlabs/logspout:latest .
