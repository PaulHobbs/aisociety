#!/bin/bash
OPENROUTER_API_KEY=$(jq -r '.OPENROUTER_API_KEY' .secrets.json) docker-compose -f docker-compose.yml -f docker-compose.test.yml $@
