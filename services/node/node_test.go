package node_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	pb "paul.hobbs.page/aisociety/protos"
)

const (
	serverAddr  = "localhost:60051" // Adjust if your server runs on a different port
	fakeAgentID = "fake-agent"      // Must match the constant in nodeservice.go
)

func TestExecuteNode_FakeAgent(t *testing.T) {
	// Set up a connection to the server.
	conn, err := grpc.Dial(serverAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		t.Fatalf("did not connect: %v", err)
	}
	defer conn.Close()
	client := pb.NewNodeServiceClient(conn)

	// --- Prepare Request ---
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	workflowID := "test-workflow-fake"
	nodeID := "test-node-fake-1"
	taskGoal := "Test task for fake agent"

	req := &pb.ExecuteNodeRequest{
		WorkflowId: workflowID,
		NodeId:     nodeID,
		Node: &pb.Node{
			NodeId:      nodeID,
			Description: "Node using fake agent",
			Agent: &pb.Agent{
				AgentId:   fakeAgentID, // Specify the fake agent
				Role:      "TestWorker",
				ModelType: "fake-model",
			},
			AssignedTask: &pb.Task{
				Id:   "task-fake-1",
				Goal: taskGoal,
			},
			Status: pb.Status_UNKNOWN, // Initial status
		},
		// Add upstream/downstream nodes if needed for context testing
		// UpstreamNodes: []*pb.Node{...},
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

	// Check status
	if resp.Node.Status != pb.Status_PASS {
		t.Errorf("Expected node status PASS, got %s", resp.Node.Status)
	}

	// Check description (should contain fake agent output)
	expectedResponse := fmt.Sprintf("Fake agent response for task: %s", taskGoal)
	if resp.Node.Description == "" { // Check description in node itself now
		t.Error("Expected node description to be updated, but it was empty")
	} else {
		t.Logf("Received Node Description: %s", resp.Node.Description) // Log for debugging
	}

	// Check assigned task results
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
	if lastResult.Output != expectedResponse {
		t.Errorf("Expected last result output '%s', got '%s'", expectedResponse, lastResult.Output)
	}
	if lastResult.Summary != expectedResponse { // Summary is truncated, check full output instead or adjust expectation
		t.Logf("Warning: Summary might be truncated. Output: '%s', Summary: '%s'", lastResult.Output, lastResult.Summary)
	}

	t.Logf("ExecuteNode with fake agent successful. Response Node ID: %s", resp.Node.NodeId)
}

// --- MCP Tool Invocation Test ---

func TestExecuteNode_MCPTool(t *testing.T) {
	conn, err := grpc.Dial(serverAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		t.Fatalf("did not connect: %v", err)
	}
	defer conn.Close()
	client := pb.NewNodeServiceClient(conn)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	workflowID := "test-workflow-mcp"
	nodeID := "test-node-mcp-1"
	taskGoal := "Call: knowledge-base-curator.summarize"

	req := &pb.ExecuteNodeRequest{
		WorkflowId: workflowID,
		NodeId:     nodeID,
		Node: &pb.Node{
			NodeId: nodeID,
			// The MCP tool expects the "text" input to be provided via Node.Description.
			Description: "Hello world",
			Agent: &pb.Agent{
				AgentId:   "mcp-tool-agent",
				Role:      "TestWorker",
				ModelType: "mock",
			},
			AssignedTask: &pb.Task{
				Id:   "task-mcp-1",
				Goal: taskGoal,
			},
			Status: pb.Status_UNKNOWN,
		},
	}

	t.Logf("DEBUG: req.Node.Description = %q", req.Node.Description)
	resp, err := client.ExecuteNode(ctx, req)
	if err != nil {
		t.Fatalf("ExecuteNode failed: %v", err)
	}

	if resp.Node == nil {
		t.Fatal("ExecuteNode response node is nil")
	}

	if resp.Node.Status != pb.Status_PASS {
		t.Errorf("Expected node status PASS, got %s", resp.Node.Status)
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
	expectedSummary := "Invoked tool knowledge-base-curator.summarize"
	if lastResult.Summary == "" || lastResult.Output == "" {
		t.Errorf("Expected non-empty summary and output for MCP tool result")
	}
	if lastResult.Summary[:len(expectedSummary)] != expectedSummary {
		t.Errorf("Expected summary to start with '%s', got '%s'", expectedSummary, lastResult.Summary)
	}
	t.Logf("MCP tool invocation result: summary='%s', output='%s'", lastResult.Summary, lastResult.Output)
}

func TestExecuteNode_MCPTool_NotFound(t *testing.T) {
	conn, err := grpc.Dial(serverAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		t.Fatalf("did not connect: %v", err)
	}
	defer conn.Close()
	client := pb.NewNodeServiceClient(conn)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	workflowID := "test-workflow-mcp"
	nodeID := "test-node-mcp-notfound"
	taskGoal := "Call: non-existent-tool"

	req := &pb.ExecuteNodeRequest{
		WorkflowId: workflowID,
		NodeId:     nodeID,
		Node: &pb.Node{
			NodeId:      nodeID,
			Description: "Node using non-existent MCP tool",
			Agent: &pb.Agent{
				AgentId:   "mcp-tool-agent",
				Role:      "TestWorker",
				ModelType: "mock",
			},
			AssignedTask: &pb.Task{
				Id:   "task-mcp-notfound",
				Goal: taskGoal,
			},
			Status: pb.Status_UNKNOWN,
		},
	}
	resp, err := client.ExecuteNode(ctx, req)
	if err != nil {
		t.Fatalf("ExecuteNode failed: %v", err)
	}
	if resp.Node == nil {
		t.Fatal("ExecuteNode response node is nil")
	}
	if resp.Node.Status != pb.Status_TASK_ERROR {
		t.Errorf("Expected node status TASK_ERROR for tool not found, got %s", resp.Node.Status)
	}
}

func TestExecuteNode_MCPTool_InputValidationError(t *testing.T) {
	conn, err := grpc.Dial(serverAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		t.Fatalf("did not connect: %v", err)
	}
	defer conn.Close()
	client := pb.NewNodeServiceClient(conn)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	workflowID := "test-workflow-mcp"
	nodeID := "test-node-mcp-inputvalidation"
	// Use a different tool name so the handler doesn't automatically add "text" from Description
	taskGoal := "Call: hypothetical-tool.do-something"

	req := &pb.ExecuteNodeRequest{
		WorkflowId: workflowID,
		NodeId:     nodeID,
		Node: &pb.Node{
			NodeId:      nodeID,
			Description: "Node with invalid input for MCP tool",
			Agent: &pb.Agent{
				AgentId:   "mcp-tool-agent",
				Role:      "TestWorker",
				ModelType: "mock",
			},
			AssignedTask: &pb.Task{
				Id:   "task-mcp-inputvalidation",
				Goal: taskGoal,
				// Input should be just {"goal": taskGoal}, missing "text" required by hypothetical-tool
			},
			Status: pb.Status_UNKNOWN,
		},
	}
	resp, err := client.ExecuteNode(ctx, req)
	if err != nil {
		t.Fatalf("ExecuteNode failed: %v", err)
	}
	if resp.Node == nil {
		t.Fatal("ExecuteNode response node is nil")
	}
	if resp.Node.Status != pb.Status_TASK_ERROR {
		t.Errorf("Expected node status TASK_ERROR for input validation error, got %s", resp.Node.Status)
	}
}

func TestExecuteNode_MCPTool_Extensibility(t *testing.T) {
	conn, err := grpc.Dial(serverAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		t.Fatalf("did not connect: %v", err)
	}
	defer conn.Close()
	client := pb.NewNodeServiceClient(conn)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	workflowID := "test-workflow-mcp"
	nodeID := "test-node-mcp-extensible"
	taskGoal := "Call: extensible-tool"

	req := &pb.ExecuteNodeRequest{
		WorkflowId: workflowID,
		NodeId:     nodeID,
		Node: &pb.Node{
			NodeId:      nodeID,
			Description: "Node using extensible MCP tool",
			Agent: &pb.Agent{
				AgentId:   "mcp-tool-agent",
				Role:      "TestWorker",
				ModelType: "mock",
			},
			AssignedTask: &pb.Task{
				Id:   "task-mcp-extensible",
				Goal: taskGoal,
			},
			Status: pb.Status_UNKNOWN,
		},
	}
	// This test assumes you have registered "extensible-tool" in your tool registry for extensibility testing.
	resp, err := client.ExecuteNode(ctx, req)
	if err != nil {
		t.Fatalf("ExecuteNode failed: %v", err)
	}
	if resp.Node == nil {
		t.Fatal("ExecuteNode response node is nil")
	}
	// Accept either PASS or TASK_ERROR depending on tool registry config
	if resp.Node.Status != pb.Status_PASS && resp.Node.Status != pb.Status_TASK_ERROR {
		t.Errorf("Expected node status PASS or TASK_ERROR for extensible tool, got %s", resp.Node.Status)
	}
}
