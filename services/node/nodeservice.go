package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"google.golang.org/protobuf/proto"
	pb "paul.hobbs.page/aisociety/protos"
)

func (s *nodeServiceServer) ExecuteNode(ctx context.Context, req *pb.ExecuteNodeRequest) (*pb.ExecuteNodeResponse, error) {
	fmt.Printf("ExecuteNode request received for node: %s in workflow: %s\n", req.GetNodeId(), req.GetWorkflowId())

	node := req.GetNode()
	if node == nil {
		return nil, fmt.Errorf("ExecuteNode: node not provided in request")
	}

	agent := node.GetAgent()
	task := node.GetAssignedTask()

	// Construct prompt from task goal and upstream context
	prompt := "Task: " + task.GetGoal() + "\n"
	prompt += "Context from dependencies:\n"
	for _, upstream := range req.GetUpstreamNodes() {
		prompt += "- " + upstream.GetAssignedTask().GetGoal() + "\n"
	}

	// Prepare OpenRouter API call
	apiURL := "https://openrouter.ai/api/v1/chat/completions"
	apiKey := os.Getenv("OPENROUTER_API_KEY")
	if apiKey == "" {
		return nil, fmt.Errorf("missing OpenRouter API key")
	}

	payload := fmt.Sprintf(`{
		"model": "%s",
		"messages": [{"role": "user", "content": "%s"}]
	}`, agent.GetModelType(), prompt)

	reqBody := strings.NewReader(payload)
	httpReq, err := http.NewRequestWithContext(ctx, "POST", apiURL, reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to create OpenRouter request: %v", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+apiKey)

	client := &http.Client{}
	resp, err := client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("OpenRouter API call failed: %v", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read OpenRouter response: %v", err)
	}

	// Parse response (simplified, assumes OpenAI-compatible JSON)
	var result struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("failed to parse OpenRouter response: %v", err)
	}

	var aiResponse string
	if len(result.Choices) > 0 {
		aiResponse = result.Choices[0].Message.Content
	} else {
		aiResponse = "No response from AI agent"
	}

	// Clone the input node to preserve all fields
	updatedNode := proto.Clone(node).(*pb.Node)

	// Update status and description
	updatedNode.Status = pb.Status_PASS
	updatedNode.Description = aiResponse

	// Prepare updated assigned_task with new result
	assignedTask := updatedNode.GetAssignedTask()
	if assignedTask == nil {
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
