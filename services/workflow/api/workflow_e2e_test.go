package api

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	pb "paul.hobbs.page/aisociety/protos"

	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

func TestWorkflowLifecycle_E2E(t *testing.T) {
	// Set up authentication tokens for the test
	os.Setenv("WORKFLOW_API_TOKENS", "admin:admin-token,user:user-token")

	target := os.Getenv("WORKFLOW_TARGET")
	if target == "" {
		target = "localhost:60052"
	}
	conn, err := grpc.Dial(target, grpc.WithInsecure(), grpc.WithBlock(), grpc.WithTimeout(5*time.Second))
	if err != nil {
		t.Fatalf("Failed to connect to WorkflowService: %v", err)
	}
	defer conn.Close()

	client := pb.NewWorkflowServiceClient(conn)
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// Add authentication metadata to context
	ctx = metadata.AppendToOutgoingContext(ctx, "authorization", "Bearer admin-token")

	// Generate UUIDs for nodes
	aID := uuid.New().String()
	bID := uuid.New().String()
	cID := uuid.New().String()

	// Create a sample workflow with 3 nodes: A -> B -> C
	createResp, err := client.CreateWorkflow(ctx, &pb.CreateWorkflowRequest{
		Nodes: []*pb.Node{
			{
				NodeId:      aID,
				Description: "Node A",
				ParentIds:   nil,
				Status:      pb.Status_BLOCKED,
			},
			{
				NodeId:      bID,
				Description: "Node B",
				ParentIds:   []string{aID},
				Status:      pb.Status_BLOCKED,
			},
			{
				NodeId:      cID,
				Description: "Node C",
				ParentIds:   []string{bID},
				Status:      pb.Status_BLOCKED,
			},
		},
	})
	if err != nil {
		t.Fatalf("CreateWorkflow failed: %v", err)
	}
	workflowID := createResp.WorkflowId

	// Poll workflow status until completion
	var nodeStatuses map[string]int32
pollLoop:
	for start := time.Now(); time.Since(start) < 30*time.Second; {
		time.Sleep(500 * time.Millisecond)

		getResp, err := client.GetWorkflow(ctx, &pb.GetWorkflowRequest{WorkflowId: workflowID})
		if err != nil {
			t.Fatalf("GetWorkflow failed: %v", err)
		}
		nodeStatuses = make(map[string]int32)
		for _, n := range getResp.Nodes {
			nodeStatuses[n.NodeId] = int32(n.Status)
		}

		allDone := true
		for _, status := range nodeStatuses {
			if status != int32(pb.Status_PASS) && status != int32(pb.Status_FAIL) {
				allDone = false
			}
		}
		if allDone {
			break pollLoop
		}
	}

	// Assert all nodes reached terminal state PASS or FAIL
	for id, status := range nodeStatuses {
		if status != int32(pb.Status_PASS) && status != int32(pb.Status_FAIL) {
			t.Errorf("Node %s did not reach terminal state, got %v", id, status)
		}
	}

	// Assert node status transitions
	for id, status := range nodeStatuses {
		if status != int32(pb.Status_PASS) {
			t.Errorf("Node %s did not reach PASS status, got %v", id, status)
		}
	}

	// Simulate failure scenario by updating node status to FAIL
	_, err = client.UpdateNode(ctx, &pb.UpdateNodeRequest{
		WorkflowId: workflowID,
		Node: &pb.Node{
			NodeId: bID,
			Status: pb.Status_FAIL,
		},
	})
	if err != nil {
		t.Fatalf("UpdateNode failed: %v", err)
	}

	// Verify node B is marked as FAIL
	getNodeResp, err := client.GetNode(ctx, &pb.GetNodeRequest{
		WorkflowId: workflowID,
		NodeId:     bID,
	})
	if err != nil {
		t.Fatalf("GetNode failed: %v", err)
	}
	if getNodeResp.Node.Status != pb.Status_FAIL {
		t.Errorf("Expected node B status FAIL, got %v", getNodeResp.Node.Status)
	}
}
