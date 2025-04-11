# Scheduler Service Discrepancy Analysis Report

## Objective

Analyze `docker-compose.yml` and `Makefile` to determine if the absence of a 'scheduler' service definition in `docker-compose.yml` is intentional or an oversight, given that the `Makefile` references stopping/removing such a service.

## Files Analyzed

*   `docker-compose.yml`
*   `Makefile`
*   `scripts/start_scheduler.sh`

## Findings

1.  **`docker-compose.yml`:** This file defines services for `postgres`, `node-service`, `workflow-service`, and `postgres-test`. It **does not** contain a service definition for `scheduler`.
2.  **`Makefile`:**
    *   Contains targets (`stop-e2e-services`, `rm-e2e-services`, `cleanup-e2e-services`) that explicitly attempt to run `docker-compose stop scheduler` and `docker-compose rm -f scheduler`.
    *   Contains a target `start-scheduler` that executes the `scripts/start_scheduler.sh` script.
    *   Contains a target `bin/scheduler_runner` to build the scheduler executable from source (`services/workflow/cmd_scheduler/main.go`).
3.  **`scripts/start_scheduler.sh`:** This script runs the compiled `./bin/scheduler_runner` executable directly on the host machine using `nohup`. It connects the scheduler to the *test* database (`postgres://aisociety:aisociety@localhost:55433/aisociety_test_db`) and does **not** use Docker Compose to start the service.

## Conclusion & Acceptance Criteria

*   **Confirm whether a 'scheduler' service should exist in `docker-compose.yml`:** Based *only* on the current execution flow shown in the `Makefile`'s `start-scheduler` target (which uses the script for testing), the service definition **should not** exist in `docker-compose.yml`. The scheduler is currently intended to be run as a standalone binary for this specific testing workflow.
*   **If it should, specify what its configuration might be:** N/A based on the above point for the *current* testing setup.
*   **If it should not, explain why the Makefile references it:** The `Makefile` references stopping/removing a `scheduler` Docker Compose service likely due to **legacy code, oversight, or incomplete refactoring**. The `docker-compose stop/rm scheduler` commands are inconsistent with how the scheduler is actually started via the `start-scheduler` script and will fail as Docker Compose does not manage a service named `scheduler`.

**Summary:** The absence of the `scheduler` service in `docker-compose.yml` is consistent with how it's started via the script for testing, but the `Makefile` contains outdated/incorrect commands attempting to manage it via Docker Compose.