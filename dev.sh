#!/bin/bash
set -euo pipefail

# Requires internet
# docker compose -f docker-compose.yml.dev up

docker compose up -d timescale
docker compose up -d api
