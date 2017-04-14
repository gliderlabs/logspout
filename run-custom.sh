set -ex

./build-custom.sh
docker build --file Dockerfile.custom -t mylogspouter .
docker run --rm --name=logspout \
    -v=/var/run/docker.sock:/var/run/docker.sock \
    -p 8000:80 \
    mylogspouter \
    ${SYSLOG}
