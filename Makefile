PROTO_SRC=protos/workflow_node.proto

.PHONY: all proto build

all: proto build

proto:
	protoc --go_out=. --go_opt=paths=source_relative \
	       --go-grpc_out=. --go-grpc_opt=paths=source_relative $(PROTO_SRC)

build:
	go build ./...