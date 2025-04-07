package main

import (
	"fmt"
	"net"

	pb "paul.hobbs.page/aisociety/protos"
	"paul.hobbs.page/aisociety/services/node"

	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

func main() {
	lis, err := net.Listen("tcp", ":50051")
	if err != nil {
		panic(fmt.Sprintf("failed to listen: %v", err))
	}
	s := grpc.NewServer()
	pb.RegisterNodeServiceServer(s, &node.Server{}) // Register our service

	// Enable reflection for debugging and testing (optional but recommended)
	reflection.Register(s)

	fmt.Println("NodeService server listening on port 50051")
	if err := s.Serve(lis); err != nil {
		panic(fmt.Sprintf("failed to serve: %v", err))
	}
}
