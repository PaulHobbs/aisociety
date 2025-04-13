#!/bin/sh
# Entrypoint script for the node service

# Check if OPENROUTER_API_KEY is already set in the environment
if [ -n "$OPENROUTER_API_KEY" ]; then
  echo "OPENROUTER_API_KEY loaded from environment variable."
else
  # Check if .secrets.json exists
  if [ -f "/app/.secrets.json" ]; then
    # Read OPENROUTER_API_KEY from .secrets.json
    API_KEY=$(jq -r '.OPENROUTER_API_KEY' /app/.secrets.json)
    if [ -n "$API_KEY" ]; then
      export OPENROUTER_API_KEY="$API_KEY"
      echo "OPENROUTER_API_KEY loaded from .secrets.json."
    else
      echo "WARNING: OPENROUTER_API_KEY not found in .secrets.json and not set as environment variable."
      echo "         Please set OPENROUTER_API_KEY in the environment or add it to /app/.secrets.json."
    fi
  else
    echo "WARNING: .secrets.json not found in /app/ and OPENROUTER_API_KEY not set as environment variable."
    echo "         Please set OPENROUTER_API_KEY in the environment or create /app/.secrets.json with the key."
  fi
fi

# Start the application
exec "$@"