#!/bin/bash
VERSION=`cat VERSION`
echo $VERSION

# us-east-1
eval $(docker run --rm -i -e AWS_ACCESS_KEY_ID=$AWS_ACCESS_KEY_ID -e AWS_SECRET_ACCESS_KEY=$AWS_SECRET_ACCESS_KEY quay.io/keboola/aws-cli ecr get-login --region us-east-1)
docker tag logspout:$VERSION 147946154733.dkr.ecr.us-east-1.amazonaws.com/keboola/logspout:$TRAVIS_TAG
docker push 147946154733.dkr.ecr.us-east-1.amazonaws.com/keboola/logspout:$TRAVIS_TAG

# eu-central-1
eval $(docker run --rm -i -e AWS_ACCESS_KEY_ID=$AWS_ACCESS_KEY_ID -e AWS_SECRET_ACCESS_KEY=$AWS_SECRET_ACCESS_KEY quay.io/keboola/aws-cli ecr get-login --region eu-central-1)
docker tag logspout:$VERSION 147946154733.dkr.ecr.eu-central-1.amazonaws.com/keboola/logspout:$TRAVIS_TAG
docker push 147946154733.dkr.ecr.eu-central-1.amazonaws.com/keboola/logspout:$TRAVIS_TAG

# ap-southeast-2
eval $(docker run --rm -i -e AWS_ACCESS_KEY_ID=$AWS_ACCESS_KEY_ID -e AWS_SECRET_ACCESS_KEY=$AWS_SECRET_ACCESS_KEY quay.io/keboola/aws-cli ecr get-login --region ap-southeast-2)
docker tag logspout:$VERSION 147946154733.dkr.ecr.ap-southeast-2.amazonaws.com/keboola/logspout:$TRAVIS_TAG
docker push 147946154733.dkr.ecr.ap-southeast-2.amazonaws.com/keboola/logspout:$TRAVIS_TAG
