#!/bin/bash

DB_URL="postgres://aisociety:aisociety@localhost:5433/aisociety_test_db?sslmode=disable"

echo "Starting scheduler_runner..."
nohup env DATABASE_URL="$DB_URL" ./bin/scheduler_runner > scheduler.log 2>&1 &
echo $! > scheduler.pid

sleep 1

if ps -p $(cat scheduler.pid) > /dev/null; then
    echo "scheduler_runner started successfully with PID $(cat scheduler.pid)"
else
    echo "Failed to start scheduler_runner. Check scheduler.log for details."
    rm -f scheduler.pid
    exit 1
fi