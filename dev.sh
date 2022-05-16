#!/bin/bash

# Requires internet
# docker-compose -f docker-compose.yml.dev up

docker-compose -f docker-compose.yml.dev up -d flower-mongo
docker-compose -f docker-compose.yml.dev up -d flower-python
docker-compose -f docker-compose.yml.dev up -d flower-node