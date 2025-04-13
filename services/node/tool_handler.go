package node

import (
	"context"
	"fmt"
	"log"
	"strings"

	"google.golang.org/protobuf/proto"
	pb "paul.hobbs.page/aisociety/protos"
)

// ToolHandler defines the interface for handling tool execution within a node.
type ToolHandler interface {
	// HandleToolExecution attempts to execute a tool based on the node's task.
	// It returns the potentially updated node, a boolean indicating if a tool was
	// handled (true) or if normal agent execution should proceed (false),
	// and an error if the handler itself failed unexpectedly.
	// If a tool is identified but fails (e.g., validation, auth, execution error),
	// the node status/description should be updated, handled should be true,
	// and error should be nil (as the failure is part of the expected workflow).
	HandleToolExecution(ctx context.Context, node *pb.Node, agent *pb.Agent, task *pb.Task) (updatedNode *pb.Node, handled bool, err error)
}

// mcpToolHandler implements the ToolHandler interface for MCP tools.
type mcpToolHandler struct {
	registry MCPToolRegistry
	adapter  MCPToolAdapter
}

// NewMCPToolHandler creates a new handler with the given registry and adapter.
func NewMCPToolHandler(registry MCPToolRegistry, adapter MCPToolAdapter) ToolHandler {
	// In a real application, registry and adapter would likely be injected
	// dependencies, potentially configured externally.
	if registry == nil {
		// Provide a default in-memory registry if none is given
		registry = NewInMemoryToolRegistry([]MCPTool{
			{
				Name:         "knowledge-base-curator.summarize",
				Description:  "Summarizes knowledge base entries",
				InputSchema:  map[string]string{"text": "string"},
				OutputSchema: map[string]string{"summary": "string"},
			},
			{
				Name:         "hypothetical-tool.do-something",
				Description:  "A tool that requires text input",
				InputSchema:  map[string]string{"text": "string"}, // Requires "text"
				OutputSchema: map[string]string{"result": "string"},
			},
		})
	}
	if adapter == nil {
		// Provide a default mock adapter if none is given
		adapter = &MockToolAdapter{}
	}
	return &mcpToolHandler{
		registry: registry,
		adapter:  adapter,
	}
}

// HandleToolExecution implements the ToolHandler interface.
func (h *mcpToolHandler) HandleToolExecution(ctx context.Context, node *pb.Node, agent *pb.Agent, task *pb.Task) (*pb.Node, bool, error) {
	goal := task.GetGoal()
	if !strings.HasPrefix(goal, "Call: ") {
		// Not a tool call, let the agent handle it.
		return node, false, nil
	}

	toolName := strings.TrimSpace(strings.TrimPrefix(goal, "Call: "))
	tool, err := h.registry.GetTool(toolName)
	if err != nil {
		// Tool not found, update node status and return handled=true
		updatedNode := h.updateNodeWithError(node, task, pb.Status_TASK_ERROR, fmt.Sprintf("MCP tool not found: %s", toolName))
		log.Printf("Tool not found: %s for node %s", toolName, node.GetNodeId())
		return updatedNode, true, nil
	}

	// --- Input Preparation ---
	// TODO: Implement a more robust way to map task/node data to tool inputs.
	// This is a placeholder based on the original logic.
	input := map[string]interface{}{
		"goal": goal, // Pass the original goal
	}
	// Specific logic for knowledge-base-curator.summarize
	if tool.Name == "knowledge-base-curator.summarize" {
		inputText := node.GetDescription() // Attempt to get text from node description
		if inputText == "" {
			// Fallback or default if description is empty - consider if this is desired
			inputText = "No description provided in node." // More informative default
			log.Printf("WARN: No description found for node %s, using default text for tool %s", node.GetNodeId(), tool.Name)
		}
		input["text"] = inputText
	}
	// Add logic here to map inputs for other tools based on their schema and node/task data.
	log.Printf("DEBUG: Prepared input for tool %s: %+v (NodeID: %s)", tool.Name, input, node.GetNodeId())

	// --- Input Validation ---
	if err := h.validateInputAgainstSchema(input, tool.InputSchema); err != nil {
		errMsg := fmt.Sprintf("Input validation error: %v", err)
		updatedNode := h.updateNodeWithError(node, task, pb.Status_TASK_ERROR, errMsg)
		log.Printf("[AUDIT] Tool invocation failed input validation: agent=%s tool=%s error=%v (NodeID: %s)", agent.GetAgentId(), tool.Name, err, node.GetNodeId())
		return updatedNode, true, nil
	}

	// --- Authorization ---
	if err := h.checkToolAuthorization(agent, tool); err != nil {
		errMsg := fmt.Sprintf("Authorization error: %v", err)
		updatedNode := h.updateNodeWithError(node, task, pb.Status_TASK_ERROR, errMsg)
		log.Printf("[AUDIT] Tool invocation failed authorization: agent=%s tool=%s error=%v (NodeID: %s)", agent.GetAgentId(), tool.Name, err, node.GetNodeId())
		return updatedNode, true, nil
	}

	// --- Tool Invocation ---
	log.Printf("[AUDIT] Invoking MCP tool: agent=%s tool=%s input=%v (NodeID: %s)", agent.GetAgentId(), tool.Name, input, node.GetNodeId())
	result, invokeErr := h.adapter.Invoke(ctx, *tool, input)

	// --- Result Handling ---
	updatedNode := proto.Clone(node).(*pb.Node)
	assignedTask := updatedNode.GetAssignedTask() // Should exist, checked earlier
	if assignedTask == nil {                      // Defensive check
		assignedTask = &pb.Task{}
		updatedNode.AssignedTask = assignedTask
		log.Printf("WARN: AssignedTask was nil unexpectedly for node %s during tool result handling", node.GetNodeId())
	}

	newResult := &pb.Task_Result{
		// Initialize common fields
		Artifacts: map[string]string{}, // Placeholder
	}

	if invokeErr != nil {
		// Tool execution failed
		newResult.Status = pb.Status_TASK_ERROR
		newResult.Summary = fmt.Sprintf("MCP tool error: %v", invokeErr)
		newResult.Output = "" // No successful output
		updatedNode.Status = pb.Status_TASK_ERROR
		updatedNode.Description = newResult.Summary
		log.Printf("[AUDIT] MCP tool invocation error: agent=%s tool=%s error=%v (NodeID: %s)", agent.GetAgentId(), tool.Name, invokeErr, node.GetNodeId())
	} else {
		// Tool execution succeeded
		newResult.Status = pb.Status_PASS
		// Safely extract summary and output from result map
		summaryStr := "Summary not found in tool result."
		if summaryVal, ok := result["summary"]; ok {
			summaryStr = fmt.Sprintf("%v", summaryVal)
		}
		outputStr := "Output not found in tool result."
		if outputVal, ok := result["output"]; ok {
			outputStr = fmt.Sprintf("%v", outputVal)
		}

		newResult.Summary = summaryStr
		newResult.Output = outputStr
		updatedNode.Status = pb.Status_PASS
		updatedNode.Description = fmt.Sprintf("MCP tool (%s) completed task. %s", toolName, truncate(newResult.Summary, 200))
		log.Printf("[AUDIT] MCP tool invocation succeeded: agent=%s tool=%s result=%v (NodeID: %s)", agent.GetAgentId(), tool.Name, result, node.GetNodeId())
	}

	assignedTask.Results = append(assignedTask.Results, newResult)
	return updatedNode, true, nil
}

// validateInputAgainstSchema checks if the input map conforms to the tool's schema.
func (h *mcpToolHandler) validateInputAgainstSchema(input map[string]interface{}, schema map[string]string) error {
	log.Printf("DEBUG: validateInputAgainstSchema input=%+v schema=%+v", input, schema)
	for key, expectedType := range schema {
		value, ok := input[key]
		if !ok {
			return fmt.Errorf("missing required input field: %s", key)
		}

		// Basic type checking - can be expanded
		switch expectedType {
		case "string":
			if _, ok := value.(string); !ok {
				return fmt.Errorf("input field '%s' must be a string, got %T", key, value)
			}
		case "int":
			// Consider allowing float64 and converting if appropriate, JSON unmarshals numbers as float64
			switch value.(type) {
			case int:
				// ok
			case float64:
				// Allow float64 if it's a whole number? Or require strict int?
				// For now, strict check.
				return fmt.Errorf("input field '%s' must be an integer, got float64", key)
			default:
				return fmt.Errorf("input field '%s' must be an integer, got %T", key, value)
			}
		// Add more types (bool, float, list, map) as needed
		default:
			return fmt.Errorf("unsupported type '%s' in tool input schema for key '%s'", expectedType, key)
		}
	}
	// Optional: Check for extra fields in input not defined in schema?
	return nil
}

// checkToolAuthorization verifies if the agent is permitted to use the tool.
func (h *mcpToolHandler) checkToolAuthorization(agent *pb.Agent, tool *MCPTool) error {
	// TODO: Implement real authorization logic based on agent roles, permissions, etc.
	// This might involve checking against an ACL, querying a permissions service, etc.
	log.Printf("DEBUG: Performing authorization check for agent %s on tool %s (currently allows all)", agent.GetAgentId(), tool.Name)
	// Placeholder: Allow all for now.
	return nil
}

// updateNodeWithError is a helper to consistently update node status and description on error.
func (h *mcpToolHandler) updateNodeWithError(originalNode *pb.Node, task *pb.Task, status pb.Status, errMsg string) *pb.Node {
	updatedNode := proto.Clone(originalNode).(*pb.Node)
	updatedNode.Status = status
	updatedNode.Description = errMsg // Set description to the error message

	// Ensure task and results array exist before appending
	assignedTask := updatedNode.GetAssignedTask()
	if assignedTask == nil {
		assignedTask = &pb.Task{} // Create task if missing
		updatedNode.AssignedTask = assignedTask
		log.Printf("WARN: AssignedTask was nil unexpectedly for node %s during error update", originalNode.GetNodeId())
	}
	// Append a result indicating the error
	errorResult := &pb.Task_Result{
		Status:  status,
		Summary: errMsg,
		Output:  "", // No output on error
	}
	assignedTask.Results = append(assignedTask.Results, errorResult)

	return updatedNode
}

// Note: The 'truncate' function is defined in service.go and used there for agent responses.
// If it were only used for tool results, it could be moved here or into a shared utility package.
