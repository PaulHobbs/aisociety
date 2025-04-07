package persistence

import (
	"context"
	"os"
	"testing"

	pb "paul.hobbs.page/aisociety/protos"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v4"
	"google.golang.org/protobuf/proto"
)

var testManager *PostgresStateManager

func TestMain(m *testing.M) {
	connStr := os.Getenv("TEST_DATABASE_URL")
	if connStr == "" {
		panic("TEST_DATABASE_URL not set")
	}

	ctx := context.Background()
	var err error
	testManager, err = NewPostgresStateManager(ctx, connStr)
	if err != nil {
		panic(err)
	}
	defer testManager.Close()

	code := m.Run()
	os.Exit(code)
}

func cleanDB(t *testing.T) {
	connStr := os.Getenv("TEST_DATABASE_URL")
	conn, err := pgx.Connect(context.Background(), connStr)
	if err != nil {
		t.Fatalf("failed to connect for cleanup: %v", err)
	}
	defer conn.Close(context.Background())

	_, err = conn.Exec(context.Background(), "TRUNCATE node_edges, nodes, workflows RESTART IDENTITY CASCADE;")
	if err != nil {
		t.Fatalf("failed to clean db: %v", err)
	}
}

func TestCreateAndGetWorkflow(t *testing.T) {
	cleanDB(t)

	ctx := context.Background()
	workflow := &Workflow{
		Name:        "Test Workflow",
		Description: "A workflow for testing",
		Status:      pb.Status_UNKOWN,
	}

	err := testManager.CreateWorkflow(ctx, workflow)
	if err != nil {
		t.Fatalf("CreateWorkflow failed: %v", err)
	}

	got, err := testManager.GetWorkflow(ctx, workflow.ID)
	if err != nil {
		t.Fatalf("GetWorkflow failed: %v", err)
	}

	if got.Name != workflow.Name || got.Description != workflow.Description {
		t.Errorf("Got workflow %+v, want %+v", got, workflow)
	}
}

func TestCreateAndGetNode(t *testing.T) {
	cleanDB(t)

	ctx := context.Background()
	workflow := &Workflow{
		Name:        "Node Test Workflow",
		Description: "Workflow for node test",
		Status:      pb.Status_UNKOWN,
	}

	err := testManager.CreateWorkflow(ctx, workflow)
	if err != nil {
		t.Fatalf("CreateWorkflow failed: %v", err)
	}

	node := &pb.Node{
		NodeId:      uuid.New().String(),
		Description: "Test Node",
		// fill other fields as needed
	}

	err = testManager.CreateNode(ctx, workflow.ID, node)
	if err != nil {
		t.Fatalf("CreateNode failed: %v", err)
	}

	gotNode, err := testManager.GetNode(ctx, workflow.ID, node.NodeId)
	if err != nil {
		t.Fatalf("GetNode failed: %v", err)
	}

	if !proto.Equal(gotNode, node) {
		t.Errorf("Got node %+v, want %+v", gotNode, node)
	}
}
