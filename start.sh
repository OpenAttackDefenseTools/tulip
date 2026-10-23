#!/bin/bash
set -euo pipefail

if [ -f .env ]; then
  # shellcheck disable=SC1091
  source .env
fi

if [ -n "${FLAGID_SCRAPE:-}" ] && [ -f docker-compose-flagid.yml ]; then
  docker compose -f docker-compose-flagid.yml up
else
  docker compose up
fi
