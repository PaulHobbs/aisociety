PROTO_SRC=protos/workflow_node.proto

.PHONY: all run-node-server

all: proto build

protos/workflow_node.pb.go protos/workflow_node_grpc.pb.go: protos/workflow_node.proto
	protoc --go_out=. --go_opt=paths=source_relative \
	       --go-grpc_out=. --go-grpc_opt=paths=source_relative $<

proto: protos/workflow_node.pb.go protos/workflow_node_grpc.pb.go

GO_SOURCES := $(shell find services/ -type f -name '*.go')

bin/node_server: proto $(GO_SOURCES)
	go build -o bin/node_server services/node/server.go

build: bin/node_server

run-node-server: build
	OPENROUTER_API_KEY=$$(jq -r .OPENROUTER_API_KEY .secrets) bin/node_server