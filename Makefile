PROTO_SRC=protos/workflow_node.proto

.PHONY: run-node-server

all: proto build

protos/workflow_node.pb.go protos/workflow_node_grpc.pb.go: protos/workflow_node.proto
	protoc --go_out=. --go_opt=paths=source_relative \
	       --go-grpc_out=. --go-grpc_opt=paths=source_relative $<

proto: protos/workflow_node.pb.go protos/workflow_node_grpc.pb.go

GO_SOURCES := $(shell find services/node -type f -name '*.go')

bin/node_server: proto $(GO_SOURCES)
	go build -o bin/node_server services/node/cmd/server.go

build: bin/node_server

run-node-server: build
	OPENROUTER_API_KEY=$$(jq -r .OPENROUTER_API_KEY .secrets.json) bin/node_server

.PHONY: test-pure test-e2e check-server

check-server:
	@nc -z localhost 50051 || (echo "Server not running, starting server..." && make run-node-server & sleep 3)

test-pure: check-server
	go test -v ./services/node -short | grep -v TestE2E | grep -v "short mode"

test-e2e: check-server
	go test -v ./services/node -run '^TestE2E_'
