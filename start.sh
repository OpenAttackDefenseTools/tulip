#!/bin/bash

source .env

if [ -n "$FLAGID_SCRAPE" ]; then
  docker-compose -f docker-compose-flagid.yml up;
elif [ -n "$LOCAL_BUILD" ]; then
  docker-compose -f docker-compose-local.yml up
else
  docker-compose up
fi

