PROTO_SRC=protos/workflow_node.proto

.PHONY: run-node-server

all: proto build

protos/workflow_node.pb.go protos/workflow_node_grpc.pb.go: protos/workflow_node.proto
	protoc --go_out=. --go_opt=paths=source_relative \
	       --go-grpc_out=. --go-grpc_opt=paths=source_relative $<

proto: protos/workflow_node.pb.go protos/workflow_node_grpc.pb.go

# Platform-agnostic way to find Go source files
GO_SOURCES := $(shell go list -f '{{$$dir := .Dir}}{{range .GoFiles}}{{$$dir}}/{{.}}{{"\n"}}{{end}}' ./services/node/...)

bin/node_server: proto $(GO_SOURCES)
	go build -o bin/node_server services/node/cmd/server.go

build: bin/node_server
	go build ./...

run-node-server: build
	@python -c "import json, os, subprocess; \
f = open('.secrets.json'); key = json.load(f).get('OPENROUTER_API_KEY', ''); f.close(); \
env = os.environ.copy(); env['OPENROUTER_API_KEY'] = key; \
subprocess.Popen(['bin/node_server'], env=env)"

.PHONY: test-pure test-e2e check-server

check-server:
	@python -c "import socket, sys; s=socket.socket(); s.settimeout(1); \
err = s.connect_ex(('localhost', 50051)); s.close(); sys.exit(0 if err==0 else 1)" || (echo "Server not running, starting server..." && make run-node-server)

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

.PHONY: test-persistence-schema init-test-db-logs

init-test-db-logs:
	docker-compose down -v
	docker-compose up -d postgres-test
	sleep 5
	docker logs aisociety_postgres_test

test-persistence-schema: start-test-db
	TEST_DATABASE_URL=postgres://aisociety:aisociety@localhost:5433/aisociety_test_db?sslmode=disable go test -v ./services/workflow/persistence/schema_test.go

test: test-workflow-storage fuzz-workflow-storage test-pure test-scheduler fuzz-scheduler

.PHONY: test-scheduler fuzz-scheduler

test-scheduler:
	go test -v ./services/workflow/scheduler

fuzz-scheduler:
	go test -fuzz=Fuzz --fuzztime=2s -v ./services/workflow/scheduler