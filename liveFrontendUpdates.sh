#!/bin/bash

function log {
    echo "> $(date +%T) $*"
}

trap "exit" INT TERM
trap "removeContainer; kill 0" EXIT

tag=$(git describe --first-parent)

function removeContainer {
    log "removing old container"
    docker stop quice
    docker rm quice
}

function startContainer {
    docker build -t r.erwin.land/gnur/quice:${tag} .
    docker run \
    -d \
    --name quice \
    -p 8624:8624 \
    --env-file .env \
    r.erwin.land/gnur/quice:${tag}

}

removeContainer

startContainer

cd app
yarn serve
