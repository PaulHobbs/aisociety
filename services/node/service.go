package node

import (
	"context"
	"fmt"
	"log"

	"google.golang.org/protobuf/proto"
	pb "paul.hobbs.page/aisociety/protos"
)

type Server struct {
	pb.UnimplementedNodeServiceServer
	toolHandler ToolHandler // Added ToolHandler field
}

// NewServer creates a new Node service server.
// In a real application, dependencies like the ToolHandler would be injected.
func NewServer() *Server {
	return &Server{
		// Initialize with default MCP handler for now
		toolHandler: NewMCPToolHandler(nil, nil),
	}
}

func (s *Server) ExecuteNode(ctx context.Context, req *pb.ExecuteNodeRequest) (*pb.ExecuteNodeResponse, error) {
	fmt.Printf("ExecuteNode request received for node: %s in workflow: %s\n", req.GetNodeId(), req.GetWorkflowId())

	node := req.GetNode()
	if node == nil {
		return nil, fmt.Errorf("ExecuteNode: node not provided in request")
	}

	agent := node.GetAgent()
	if agent == nil {
		return nil, fmt.Errorf("ExecuteNode: agent not provided in node")
	}
	task := node.GetAssignedTask()
	if task == nil {
		return nil, fmt.Errorf("ExecuteNode: assigned_task not provided in node")
	}

	// --- Tool Handling Delegation ---
	// Attempt to handle the task using the configured ToolHandler
	updatedNode, handled, toolErr := s.toolHandler.HandleToolExecution(ctx, node, agent, task)
	if toolErr != nil {
		// Handle unexpected errors within the tool handler itself
		log.Printf("ERROR: Tool handler failed for node %s: %v", req.GetNodeId(), toolErr)
		// Return a generic error or update node status to reflect internal error
		// Cloning the original node to set error status
		errorNode := proto.Clone(node).(*pb.Node)
		errorNode.Status = pb.Status_TASK_ERROR
		errorNode.Description = fmt.Sprintf("Internal error during tool handling: %v", toolErr)
		// Optionally add a result to the task
		if errorNode.AssignedTask == nil {
			errorNode.AssignedTask = &pb.Task{}
		}
		errorNode.AssignedTask.Results = append(errorNode.AssignedTask.Results, &pb.Task_Result{
			Status:  pb.Status_TASK_ERROR,
			Summary: errorNode.Description,
		})
		return &pb.ExecuteNodeResponse{Node: errorNode}, nil // Return error node, but nil gRPC error
	}

	if handled {
		// Tool handler processed the request (successfully or with a handled error like validation/auth failure)
		// Return the node state as updated by the handler.
		log.Printf("Tool handler processed node %s, returning updated node.", req.GetNodeId())
		return &pb.ExecuteNodeResponse{Node: updatedNode}, nil
	}

	// --- If not handled by ToolHandler, proceed with Agent Execution ---
	log.Printf("Task for node %s not handled by tool handler, proceeding with agent execution.", req.GetNodeId())

	// --- Agent Selection Logic ---
	// Construct prompt from task goal and upstream context
	prompt := "Task: " + task.GetGoal() + "\n"
	prompt += "Context from dependencies:\n"
	for _, upstream := range req.GetUpstreamNodes() {
		if upstream != nil && upstream.GetAssignedTask() != nil {
			prompt += "- " + upstream.GetAssignedTask().GetGoal() + "\n"
		}
	}

	var aiResponse string
	var agentErr error

	switch agent.GetAgentId() {
	case FakeAgentID:
		fmt.Println("Using Fake Agent")
		aiResponse, agentErr = callFakeAgent(ctx, agent, prompt, task.GetGoal())
	default:
		fmt.Println("Using Real Agent (OpenRouter)")
		aiResponse, agentErr = callRealAgent(ctx, agent, prompt)
	}
	// --- End Agent Selection ---

	if agentErr != nil {
		// Handle agent error - maybe set node status to TASK_ERROR
		updatedNode := proto.Clone(node).(*pb.Node)
		updatedNode.Status = pb.Status_TASK_ERROR
		updatedNode.Description = fmt.Sprintf("Agent error: %v", agentErr)
		return &pb.ExecuteNodeResponse{Node: updatedNode}, nil
	}

	// Clone the input node to preserve all fields
	updatedNode = proto.Clone(node).(*pb.Node)

	// Update status and description based on successful agent execution
	updatedNode.Status = pb.Status_PASS
	updatedNode.Description = fmt.Sprintf("Agent (%s) completed task. Response: %s", agent.GetAgentId(), truncate(aiResponse, 200)) // Add agent ID to description

	// Prepare updated assigned_task with new result
	assignedTask := updatedNode.GetAssignedTask() // Already checked for nil above
	if assignedTask == nil {                      // Should not happen due to check above, but defensive coding
		assignedTask = &pb.Task{}
		updatedNode.AssignedTask = assignedTask
	}

	// Create a new result
	newResult := &pb.Task_Result{
		Status:    pb.Status_PASS,
		Summary:   truncate(aiResponse, 100),
		Output:    aiResponse,
		Artifacts: map[string]string{}, // Add any artifacts if available
	}

	// Append the new result
	assignedTask.Results = append(assignedTask.Results, newResult)

	return &pb.ExecuteNodeResponse{
		Node: updatedNode,
	}, nil
}

// truncate returns the first n characters of s, or s itself if shorter
func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n]
}
