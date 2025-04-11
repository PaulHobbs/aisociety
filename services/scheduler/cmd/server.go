package main

import (
	"context"
	"log"
	"os"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	pb "paul.hobbs.page/aisociety/protos"
	scheduler "paul.hobbs.page/aisociety/services/scheduler/internal"
	"paul.hobbs.page/aisociety/services/workflow/persistence"
)

type grpcNodeClientWrapper struct {
	client pb.NodeServiceClient
}

func (w *grpcNodeClientWrapper) ExecuteNode(ctx context.Context, req *pb.ExecuteNodeRequest) (*pb.ExecuteNodeResponse, error) {
	return w.client.ExecuteNode(ctx, req)
}

func main() {
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		log.Fatal("DATABASE_URL not set")
	}

	ctx := context.Background()
	sm, err := persistence.NewPostgresStateManagerFromConnStr(ctx, dbURL)
	if err != nil {
		log.Fatalf("failed to create state manager: %v", err)
	}

	nodeTarget := os.Getenv("NODE_TARGET")
	if nodeTarget == "" {
		nodeTarget = "localhost:50051"
	}

	conn, err := grpc.NewClient(nodeTarget, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("failed to connect to node service: %v", err)
	}
	defer conn.Close()

	nodeClient := pb.NewNodeServiceClient(conn)
	wrappedClient := &grpcNodeClientWrapper{client: nodeClient}

	sched := scheduler.NewSimpleScheduler(sm, wrappedClient, 2*time.Second)

	log.Println("Starting scheduler...")
	sched.Run(ctx)
}
