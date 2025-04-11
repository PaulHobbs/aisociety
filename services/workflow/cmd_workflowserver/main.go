package main

import (
	"context"
	"fmt"
	"net"
	"os"

	pb "paul.hobbs.page/aisociety/protos"
	"paul.hobbs.page/aisociety/services/workflow/api"
	"paul.hobbs.page/aisociety/services/workflow/persistence"

	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

func main() {
	port := os.Getenv("WORKFLOW_PORT")
	if port == "" {
		port = "50052"
	}
	addr := ":" + port

	connStr := os.Getenv("DATABASE_URL")
	if connStr == "" {
		panic("DATABASE_URL not set")
	}

	ctx := context.Background()
	sm, err := persistence.NewPostgresStateManagerFromConnStr(ctx, connStr)
	if err != nil {
		panic(fmt.Sprintf("failed to connect to DB: %v", err))
	}

	lis, err := net.Listen("tcp", addr)
	if err != nil {
		panic(fmt.Sprintf("failed to listen on %s: %v", addr, err))
	}

	s := grpc.NewServer(grpc.UnaryInterceptor(api.AuthInterceptor))

	workflowSvc := api.NewWorkflowServiceServer(sm, &api.StdoutEventLogger{})
	pb.RegisterWorkflowServiceServer(s, workflowSvc)

	reflection.Register(s)

	fmt.Printf("WorkflowService server listening on port %s\n", port)
	if err := s.Serve(lis); err != nil {
		panic(fmt.Sprintf("failed to serve: %v", err))
	}
}
