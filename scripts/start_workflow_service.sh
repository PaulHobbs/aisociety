#!/bin/bash

DB_URL="postgres://aisociety:aisociety@localhost:55433/aisociety_test_db?sslmode=disable"

echo "Starting workflow_server..."
nohup env DATABASE_URL="$DB_URL" ./bin/workflow_server > workflow_service.log 2>&1 &
echo $! > workflow_service.pid

sleep 1

if ps -p $(cat workflow_service.pid) > /dev/null; then
    echo "workflow_server started successfully with PID $(cat workflow_service.pid)"
else
    echo "Failed to start workflow_server. Check workflow_service.log for details."
    rm -f workflow_service.pid
    exit 1
fi