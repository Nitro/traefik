#!/bin/bash

die() {
    echo $1
    exit 1
}

CGO_ENABLED=0 GOOS=linux go build

file traefik | grep "ELF.*LSB" || die "../traefik is missing or not a Linux binary"

CURRENT_REVISION=$(git rev-parse --short HEAD)
docker build -t gonitro/traefik:${CURRENT_REVISION} -f DockerfileNitro . || die "Failed to build container"
docker push gonitro/traefik:${CURRENT_REVISION}
