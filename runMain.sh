#!/bin/bash
set -e
clear

tag=$(git describe --first-parent)
echo "building ${tag}"

docker build -t r.erwin.land/gnur/quice:${tag} .

if [[ "$1" == "push" ]]; then
    docker push r.erwin.land/gnur/quice:${tag}
    exit 0
fi

docker run \
    -p 8624:8624 \
    --env-file .env \
    r.erwin.land/gnur/quice:${tag}
