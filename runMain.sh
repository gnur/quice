#!/bin/bash
set -e
clear

tag=$(git describe --first-parent)

docker build -t r.erwin.land/gnur/quice:${tag} .
docker run \
    -p 8624:8624 \
    --env-file .env \
    r.erwin.land/gnur/quice:${tag}
