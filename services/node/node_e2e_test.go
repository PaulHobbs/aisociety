package node_test

import (
	"context"
	"testing"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	pb "paul.hobbs.page/aisociety/protos"
)

func TestE2E_ExecuteNode_RealAgent(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping testing in short mode")
	}

	// Set up a connection to the server.
	conn, err := grpc.Dial(serverAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		t.Fatalf("did not connect: %v", err)
	}
	defer conn.Close()
	client := pb.NewNodeServiceClient(conn)

	// --- Prepare Request ---
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*30)
	defer cancel()

	workflowID := "test-workflow-real"
	nodeID := "test-node-real-1"
	taskGoal := "Explain the theory of relativity in simple terms."

	req := &pb.ExecuteNodeRequest{
		WorkflowId: workflowID,
		NodeId:     nodeID,
		Node: &pb.Node{
			NodeId:      nodeID,
			Description: "Node using real agent",
			Agent: &pb.Agent{
				AgentId:   "openrouter/quasar-alpha", // Real agent ID
				Role:      "Explainer",
				ModelType: "openrouter/quasar-alpha",
			},
			AssignedTask: &pb.Task{
				Id:   "task-real-1",
				Goal: taskGoal,
			},
			Status: pb.Status_UNKOWN,
		},
	}

	// --- Call RPC ---
	resp, err := client.ExecuteNode(ctx, req)
	if err != nil {
		t.Fatalf("ExecuteNode failed: %v", err)
	}

	// --- Assertions ---
	if resp.Node == nil {
		t.Fatal("ExecuteNode response node is nil")
	}

	if resp.Node.Status != pb.Status_PASS {
		t.Errorf("Expected node status PASS, got %s", resp.Node.Status)
	}

	if resp.Node.Description == "" {
		t.Error("Expected node description to be updated, but it was empty")
	} else {
		t.Logf("Received Node Description: %s", resp.Node.Description)
	}

	assignedTask := resp.Node.GetAssignedTask()
	if assignedTask == nil {
		t.Fatal("Response node assigned task is nil")
	}
	if len(assignedTask.Results) == 0 {
		t.Fatal("Expected at least one result in assigned task, got none")
	}

	lastResult := assignedTask.Results[len(assignedTask.Results)-1]
	if lastResult.Status != pb.Status_PASS {
		t.Errorf("Expected last result status PASS, got %s", lastResult.Status)
	}
	if lastResult.Output == "" {
		t.Error("Expected non-empty output from real agent, got empty string")
	} else {
		t.Logf("Real agent output: %s", lastResult.Output)
	}
}
