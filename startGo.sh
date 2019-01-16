#!/bin/bash

source .env

export S3_HOST
export S3_BUCKET
export S3_ACCESS_KEY_ID
export S3_SECRET_ACCESS_KEY
export BIND_ADDR=localhost:8624
export LOGLEVEL=DEBUG

go build
./quice
