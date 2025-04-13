# Use 'docker-compose up' or 'make start-services' to run services in containers.

PROTO_SRC=protos/workflow_node.proto
TEST_DOCKER=./test-docker-compose.sh

all: proto build

protos/workflow_node.pb.go protos/workflow_node_grpc.pb.go: protos/workflow_node.proto
	protoc --go_out=. --go_opt=paths=source_relative \
	       --go-grpc_out=. --go-grpc_opt=paths=source_relative $<

proto: protos/workflow_node.pb.go protos/workflow_node_grpc.pb.go

# Platform-agnostic way to find Go source files
GO_SOURCES := $(shell go list -f '{{$$dir := .Dir}}{{range .GoFiles}}{{$$dir}}/{{.}}{{"\n"}}{{end}}' ./services/node/...)

bin/node_server: proto $(GO_SOURCES)
	go build -o bin/node_server services/node/cmd/server.go

build: bin/node_server bin/workflow_server bin/scheduler_runner
	go build ./...

bin/scheduler_runner:
	go build -o bin/scheduler_runner services/workflow/cmd_scheduler/main.go

bin/scheduler_service:
	go build -o bin/scheduler_service services/scheduler/cmd/server.go

scheduler-build: bin/scheduler_service

scheduler-test:
	go test -v ./services/scheduler/...

.PHONY: test-pure test-e2e check-server

test-pure: start-services
	go test -v ./services/node/... -short | grep -v TestE2E | grep -v "short mode"

test-e2e: start-services
	go test -v ./services/node/... -run '^TestE2E_'

.PHONY: start-test-db test-workflow-storage

start-test-db:
	@if [ "$$(docker ps -q -f name=aisociety_postgres_test)" = "" ]; then \
		echo "Starting postgres-test container..."; \
		$(TEST_DOCKER) up -d postgres-test; \
		echo "Waiting for postgres-test to initialize..."; \
		sleep 5; \
	else \
		echo "postgres-test container is already running."; \
	fi

test-workflow-storage: start-test-db
		TEST_DATABASE_URL=postgres://aisociety:aisociety@localhost:55433/aisociety_test_db?sslmode=disable go test -v ./services/workflow/persistence/...
	
fuzz-workflow-storage: start-test-db
		TEST_DATABASE_URL=postgres://aisociety:aisociety@localhost:55433/aisociety_test_db?sslmode=disable go test -fuzz=Fuzz --fuzztime=2s -v ./services/workflow/persistence
init-test-db-schema: start-test-db
	docker exec -i aisociety_postgres_test psql -U aisociety -d aisociety_test_db < services/workflow/schema/schema.sql


reset-test-db-schema: start-db
	docker exec -i aisociety_postgres psql -U aisociety -d aisociety_db -c "DROP TABLE IF EXISTS node_edges CASCADE;"
	docker exec -i aisociety_postgres psql -U aisociety -d aisociety_db -c "DROP TABLE IF EXISTS nodes CASCADE;"
	docker exec -i aisociety_postgres psql -U aisociety -d aisociety_db -c "DROP TABLE IF EXISTS workflows CASCADE;"
	docker exec -i aisociety_postgres psql -U aisociety -d aisociety_db < services/workflow/schema/schema.sql

.PHONY: test-persistence-schema

test-persistence-schema: start-test-db
	TEST_DATABASE_URL=postgres://aisociety:aisociety@localhost:55433/aisociety_test_db?sslmode=disable go test -v ./services/workflow/persistence/schema_test.go

test: test-workflow-storage fuzz-workflow-storage test-pure test-scheduler fuzz-scheduler

.PHONY: test-workflow-api
test-workflow-api:
	go test -v ./services/workflow/api/...

.PHONY: test-scheduler fuzz-scheduler

test-scheduler:
	go test -v ./services/scheduler/...

fuzz-scheduler:
.PHONY: stop-services rm-services test-e2e-workflow

start-services: $(GO_SOURCES)
	$(TEST_DOCKER) up -d --build

stop-services:
	$(TEST_DOCKER) stop node workflow scheduler postgres-test
rm-services:
	$(TEST_DOCKER) rm -f node workflow scheduler postgres-test

cleanup-services:
	@echo "Cleaning up existing containers..."
	$(TEST_DOCKER) stop node workflow scheduler postgres-test
	$(TEST_DOCKER) rm -f node workflow scheduler postgres-test

test-e2e-workflow: cleanup-services start-services
	WORKFLOW_TARGET=localhost:60052 go test -v ./services/workflow/api -run ^TestWorkflowLifecycle_E2E$
	$(MAKE) stop-services
	go test -fuzz=Fuzz --fuzztime=2s -v ./services/scheduler/internal

.PHONY: test-e2e-no-start
test-e2e-no-start:
	WORKFLOW_TARGET=localhost:60052 go test -v ./services/workflow/api -run ^TestWorkflowLifecycle_E2E$
	go test -fuzz=Fuzz --fuzztime=2s -v ./services/scheduler/internal

.PHONY: logs-workflow
logs-workflow:
	$(TEST_DOCKER) logs workflow

.PHONY: test-node-docker
test-node-docker:
	$(TEST_DOCKER) up --build --abort-on-container-exit --remove-orphans --exit-code-from node-test node-test