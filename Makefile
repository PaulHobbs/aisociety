# Use 'docker-compose up' or 'make start-e2e-services' to run services in containers.

PROTO_SRC=protos/workflow_node.proto

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

check-server:
	@python -c "import socket, sys; s=socket.socket(); s.settimeout(1); \
err = s.connect_ex(('localhost', 50051)); s.close(); sys.exit(0 if err==0 else 1)" || (echo "Server not running, starting server..." && make start-node-server)

test-pure: check-server
	go test -v ./services/node -short | grep -v TestE2E | grep -v "short mode"

test-e2e: check-server
	go test -v ./services/node -run '^TestE2E_'

.PHONY: start-test-db test-workflow-storage

start-test-db:
	@if [ "$$(docker ps -q -f name=aisociety_postgres_test)" = "" ]; then \
		echo "Starting postgres-test container..."; \
		docker-compose up -d postgres-test; \
		echo "Waiting for postgres-test to initialize..."; \
		sleep 5; \
	else \
		echo "postgres-test container is already running."; \
	fi

test-workflow-storage: start-test-db
		TEST_DATABASE_URL=postgres://aisociety:aisociety@localhost:5433/aisociety_test_db?sslmode=disable go test -v ./services/workflow/persistence/...
	
fuzz-workflow-storage: start-test-db
		TEST_DATABASE_URL=postgres://aisociety:aisociety@localhost:5433/aisociety_test_db?sslmode=disable go test -fuzz=Fuzz --fuzztime=2s -v ./services/workflow/persistence
init-test-db-schema: start-test-db
	docker exec -i aisociety_postgres_test psql -U aisociety -d aisociety_test_db < services/workflow/schema/schema.sql


reset-test-db-schema: start-test-db
	docker exec -i aisociety_postgres_test psql -U aisociety -d aisociety_test_db -c "DROP TABLE IF EXISTS node_edges CASCADE;"
	docker exec -i aisociety_postgres_test psql -U aisociety -d aisociety_test_db -c "DROP TABLE IF EXISTS nodes CASCADE;"
	docker exec -i aisociety_postgres_test psql -U aisociety -d aisociety_test_db -c "DROP TABLE IF EXISTS workflows CASCADE;"
	docker exec -i aisociety_postgres_test psql -U aisociety -d aisociety_test_db < services/workflow/schema/schema.sql

.PHONY: test-persistence-schema init-test-db-logs

init-test-db-logs:
	docker-compose down -v
	docker-compose up -d postgres-test
	sleep 5
	docker logs aisociety_postgres_test

test-persistence-schema: start-test-db
	URL=postgres://aisociety:aTEST_DATABASE_isociety@localhost:5433/aisociety_test_db?sslmode=disable go test -v ./services/workflow/persistence/schema_test.go

test: test-workflow-storage fuzz-workflow-storage test-pure test-scheduler fuzz-scheduler

.PHONY: test-scheduler fuzz-scheduler

test-scheduler:
	go test -v ./services/scheduler/...

fuzz-scheduler:
.PHONY: start-e2e-services stop-e2e-services rm-e2e-services test-e2e-workflow start-node-server

start-node-server: build
	@python -c "import json, os, subprocess; \
f = open('.secrets.json'); key = json.load(f).get('OPENROUTER_API_KEY', ''); f.close(); \
env = os.environ.copy(); env['OPENROUTER_API_KEY'] = key; \
subprocess.Popen(['bin/node_server'], env=env)"

start-e2e-services: start-test-db
	docker-compose -f docker-compose.yml -f docker-compose.test.yml up -d postgres-test
	docker-compose -f docker-compose.yml -f docker-compose.test.yml up -d node-service workflow-service scheduler

stop-e2e-services:
	docker-compose -f docker-compose.yml -f docker-compose.test.yml stop node-service workflow-service scheduler postgres-test
rm-e2e-services:
	docker-compose -f docker-compose.yml -f docker-compose.test.yml rm -f node-service workflow-service scheduler postgres-test

cleanup-e2e-services:
	@echo "Cleaning up existing containers..."
	docker-compose -f docker-compose.yml -f docker-compose.test.yml stop node-service workflow-service scheduler postgres-test
	docker-compose -f docker-compose.yml -f docker-compose.test.yml rm -f node-service workflow-service scheduler postgres-test

test-e2e-workflow: cleanup-e2e-services start-e2e-services
	go test -v ./services/workflow/api -run ^TestWorkflowLifecycle_E2E$
	$(MAKE) stop-e2e-services
	go test -fuzz=Fuzz --fuzztime=2s -v ./services/scheduler/...

.PHONY: test-e2e-no-start
test-e2e-no-start:
	go test -v ./services/workflow/api -run ^TestWorkflowLifecycle_E2E$
	go test -fuzz=Fuzz --fuzztime=2s -v ./services/scheduler/...