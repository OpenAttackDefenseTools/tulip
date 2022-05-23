#!/bin/bash

# Requires internet
# docker-compose -f docker-compose.yml.dev up

docker-compose up -d mongo
docker-compose up -d api
