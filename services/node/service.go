package node

import (
	"context"
	"fmt"

	"google.golang.org/protobuf/proto"
	pb "paul.hobbs.page/aisociety/protos"
)

type Server struct {
	pb.UnimplementedNodeServiceServer
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

	// --- Agent Selection Logic ---
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
		// For now, just return the error
		return nil, fmt.Errorf("agent execution failed: %v", agentErr)
		// TODO: Update node status to reflect error instead of returning gRPC error?
		// updatedNode := proto.Clone(node).(*pb.Node)
		// updatedNode.Status = pb.Status_TASK_ERROR
		// updatedNode.Description = fmt.Sprintf("Agent error: %v", agentErr)
		// return &pb.ExecuteNodeResponse{Node: updatedNode}, nil
	}

	// Clone the input node to preserve all fields
	updatedNode := proto.Clone(node).(*pb.Node)

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
