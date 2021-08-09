#!/bin/sh
set -ex
CGO_ENABLED=0 GOOS=linux go build -v -a -o stickyapp
