package main

import (
	"fmt"
	"net"
	"os"

	pb "paul.hobbs.page/aisociety/protos"
	"paul.hobbs.page/aisociety/services/node"

	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

func main() {
	port := os.Getenv("NODE_PORT")
	if port == "" {
		port = "50051"
	}
	addr := ":" + port

	lis, err := net.Listen("tcp", addr)
	if err != nil {
		panic(fmt.Sprintf("failed to listen on %s: %v", addr, err))
	}

	s := grpc.NewServer()
	pb.RegisterNodeServiceServer(s, &node.Server{}) // Register our service

	// Enable reflection for debugging and testing (optional but recommended)
	reflection.Register(s)

	fmt.Printf("NodeService server listening on port %s\n", port)
	if err := s.Serve(lis); err != nil {
		panic(fmt.Sprintf("failed to serve: %v", err))
	}
}
