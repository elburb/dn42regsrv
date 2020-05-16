#!/bin/bash

docker run -it -v $(dirname "$(dirname $PWD)"):/go/src/dn42regsrv golang:alpine ash -c 'apk add git && cd src/dn42regsrv && go get && cp /go/bin/dn42regsrv .'
cd ../../
docker build -t dn42regsrv -f contrib/docker/build/Dockerfile .
rm -f dn42regsrv
