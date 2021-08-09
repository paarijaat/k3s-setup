#!/bin/sh
set -ex
docker build -t stickyapp .
docker tag stickyapp:latest paarijaat/stickyapp:latest
docker push paarijaat/stickyapp:latest
docker tag stickyapp:latest paarijaat-debian-vm:5000/paarijaat/stickyapp:latest
docker push paarijaat-debian-vm:5000/paarijaat/stickyapp:latest
