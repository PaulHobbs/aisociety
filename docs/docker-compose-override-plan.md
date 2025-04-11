# Plan: Unifying Environment Management with Docker Compose Overrides

## Goal

Refactor the project's service management to consistently use Docker Compose for both standard (development/production) and testing environments, including the `scheduler` service. This eliminates the need for separate scripts like `start_scheduler.sh` for testing and corrects inconsistencies in the `Makefile`.

## Approach: Docker Compose Overrides

Utilize Docker Compose's override file mechanism (`-f` flag) to manage environment-specific configurations.

1.  **Base Configuration (`docker-compose.yml`):**
    *   Define all core services, including `postgres`, `node-service`, `workflow-service`, and the **newly added `scheduler` service**.
    *   Configure services to use the standard `postgres` database by default.
    *   Define the `postgres-test` service for testing purposes.

2.  **Test Override Configuration (`docker-compose.test.yml`):**
    *   Create a new file specifically for test environment overrides.
    *   This file will *only* contain the differences needed for testing.
    *   Override the `scheduler` service definition to:
        *   Depend on `postgres-test` instead of `postgres`.
        *   Use the `DATABASE_URL` pointing to `postgres-test`.
    *   Optionally, override `workflow-service` similarly if it needs to connect to `postgres-test` during E2E tests.

3.  **`Makefile` Updates:**
    *   Modify targets related to testing (e.g., `start-e2e-services`, `stop-e2e-services`, `rm-e2e-services`, `cleanup-e2e-services`, `test-e2e-workflow`) to use *both* compose files:
        ```bash
        docker-compose -f docker-compose.yml -f docker-compose.test.yml up -d [service...]
        docker-compose -f docker-compose.yml -f docker-compose.test.yml stop [service...]
        docker-compose -f docker-compose.yml -f docker-compose.test.yml rm -f [service...]
        ```
    *   Remove the `start-scheduler` target that calls the shell script.
    *   Remove the dependency on building `bin/scheduler_runner` directly if the Docker build handles it.
    *   Ensure the `docker-compose stop/rm` commands in the Makefile now correctly reference the `scheduler` service, as it will be defined in the base `docker-compose.yml`.

## Proposed `docker-compose.yml` Addition (Conceptual)

```yaml
# docker-compose.yml (add this service)
services:
  # ... other services ...

  scheduler:
    build:
      context: .
      # Assumes same Dockerfile as workflow-service, adjust if needed
      dockerfile: services/workflow/Dockerfile
    container_name: scheduler
    depends_on:
      - postgres # Default dependency
    environment:
      # Default DB connection
      DATABASE_URL: postgres://aisociety:aisociety@postgres:5432/aisociety_db?sslmode=disable
    # Add ports if necessary

  # ... postgres-test definition ...
```

## Proposed `docker-compose.test.yml` (New File)

```yaml
# docker-compose.test.yml
version: '3.8'

services:
  scheduler:
    depends_on:
      - postgres-test # Override dependency
    environment:
      # Override DB connection
      DATABASE_URL: postgres://aisociety:aisociety@postgres-test:55432/aisociety_test_db?sslmode=disable

  # Optional: Override workflow-service if needed for E2E tests
  # workflow-service:
  #   depends_on:
  #     - postgres-test
  #   environment:
  #     DATABASE_URL: postgres://aisociety:aisociety@postgres-test:55432/aisociety_test_db?sslmode=disable
```

## Benefits

*   Consistent environment management using a single tool (Docker Compose).
*   Clear separation of base and test configurations.
*   Removes outdated script dependencies and corrects `Makefile` inconsistencies.
*   Simplifies starting/stopping services for different environments.

## Next Steps

Once this plan is approved, switch to Code mode to implement the changes in `docker-compose.yml`, create `docker-compose.test.yml`, and update the `Makefile`.