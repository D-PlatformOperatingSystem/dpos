#!/usr/bin/env bash
# first you must build docker image, you can use make docker command
# docker build . -f Dockerfile-run -t dplatformos-build:latest

sudo docker run -it -p 28803:8801 -p 8802:8802 -p 6060:6060 -p 50051:50051 -l linux-dplatformos-run \
    -v "$GOPATH"/src/github.com/dplatformos/dplatformos:/go/src/github.com/dplatformos/dplatformos \
    -w /go/src/github.com/dplatformos/dplatformos dplatformos:latest
