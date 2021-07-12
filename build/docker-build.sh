#!/usr/bin/env bash
# https://hub.docker.com/r/suyanlong/golang-dev/
# https://github.com/suyanlong/golang-dev
# sudo docker pull suyanlong/golang-dev:latest

sudo docker run -it -p 28803:28803 -p 28804:28804 -p 6060:6060 -p 50051:50051 -l linux-dplatformos-build \
    -v "$GOPATH"/src/github.com/dplatformos/dplatformos:/go/src/github.com/dplatformos/dplatformos \
    -w /go/src/github.com/dplatformos/dplatformos suyanlong/golang-dev:latest
