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

// NewServerWithRegistry creates a new Node service server with a provided MCPToolRegistry.
func NewServerWithRegistry(registry MCPToolRegistry) *Server {
	return &Server{
		toolHandler: NewMCPToolHandler(registry, nil),
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
	// Make MCPToolRegistry available throughout the function
	var registry MCPToolRegistry
	if s.toolHandler != nil {
		registry = s.toolHandler.Registry()
	}

	updatedNode := node

	// --- MCP Tool Invocation: If task goal starts with "Call: toolname", invoke MCP tool ---
	if task != nil && len(task.Goal) > 6 && task.Goal[:5] == "Call:" {
		toolName := ""
		// Parse tool name: "Call: toolname" or "Call: toolname {...json...}"
		goalRemainder := task.Goal[5:]
		for i, c := range goalRemainder {
			if c == ' ' || c == '{' {
				toolName = goalRemainder[:i]
				break
			}
		}
		if toolName == "" {
			toolName = goalRemainder
		}
		// For now, stub: parameters are empty. (Future: parse from goal or task fields)
		inputParams := map[string]interface{}{}

		// Use MockToolAdapter for all tools for now
		adapterSelector := func(tool MCPTool) MCPToolAdapter {
			return &MockToolAdapter{}
		}
		// Use registry from toolHandler if available, else skip
		var registry MCPToolRegistry
		if s.toolHandler != nil {
			registry = s.toolHandler.Registry()
		}
		if registry != nil {
			resultMap, err := InvokeMCPTool(ctx, toolName, inputParams, registry, adapterSelector)
			updatedNode = proto.Clone(node).(*pb.Node)
			assignedTask := updatedNode.GetAssignedTask()
			if assignedTask == nil {
				assignedTask = &pb.Task{}
				updatedNode.AssignedTask = assignedTask
			}
			newResult := &pb.Task_Result{
				Artifacts: map[string]string{},
			}
			if err != nil {
				updatedNode.Status = pb.Status_TASK_ERROR
				updatedNode.Description = fmt.Sprintf("MCP tool error: %v", err)
				newResult.Status = pb.Status_TASK_ERROR
				newResult.Summary = "MCP tool invocation failed"
				newResult.Output = err.Error()
			} else {
				// Map resultMap fields to proto
				statusStr, _ := resultMap["status"].(string)
				switch statusStr {
				case "PASS":
					updatedNode.Status = pb.Status_PASS
					newResult.Status = pb.Status_PASS
				case "FAIL":
					updatedNode.Status = pb.Status_FAIL
					newResult.Status = pb.Status_FAIL
				case "TIMEOUT":
					updatedNode.Status = pb.Status_TIMEOUT
					newResult.Status = pb.Status_TIMEOUT
				default:
					updatedNode.Status = pb.Status_UNKNOWN
					newResult.Status = pb.Status_UNKNOWN
				}
				if summary, ok := resultMap["summary"].(string); ok {
					newResult.Summary = summary
				}
				if output, ok := resultMap["output"].(string); ok {
					newResult.Output = output
				}
				if artifacts, ok := resultMap["artifacts"].(map[string]string); ok {
					newResult.Artifacts = artifacts
				}
				updatedNode.Description = fmt.Sprintf("MCP tool (%s) completed. %s", toolName, newResult.Summary)
			}
			assignedTask.Results = append(assignedTask.Results, newResult)
			return &pb.ExecuteNodeResponse{Node: updatedNode}, nil
		}
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
		aiResponse, agentErr = callRealAgent(ctx, agent, prompt, registry)
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
