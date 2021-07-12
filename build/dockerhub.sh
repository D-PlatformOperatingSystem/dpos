#!/bin/bash

version=$(./dplatformos -v)
docker build . -f Dockerfile-node -t dom/node:"$version"

docker tag dom/node:"$version" dom/node:latest

docker login
docker push dom/node:latest
docker push dom/node:"$version"
