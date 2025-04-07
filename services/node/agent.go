package node

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	pb "paul.hobbs.page/aisociety/protos"
)

// Constants for agent IDs
const (
	FakeAgentID = "fake-agent"
)

type Result struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
}

// callRealAgent handles the interaction with the actual OpenRouter API
func callRealAgent(ctx context.Context, agent *pb.Agent, prompt string) (string, error) {
	apiURL := "https://openrouter.ai/api/v1/chat/completions"
	apiKey := os.Getenv("OPENROUTER_API_KEY")
	if apiKey == "" {
		return "", fmt.Errorf("missing OpenRouter API key")
	}

	payload := fmt.Sprintf(`{
		"model": "%s",
		"messages": [{"role": "user", "content": "%s"}]
	}`, agent.GetModelType(), prompt)

	reqBody := strings.NewReader(payload)
	httpReq, err := http.NewRequestWithContext(ctx, "POST", apiURL, reqBody)
	if err != nil {
		return "", fmt.Errorf("failed to create OpenRouter request: %v", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+apiKey)

	client := &http.Client{}
	resp, err := client.Do(httpReq)
	if err != nil {
		return "", fmt.Errorf("OpenRouter API call failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("OpenRouter API call returned status %d: %s", resp.StatusCode, string(respBody))
	}

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read OpenRouter response: %v", err)
	}

	// Parse response (simplified, assumes OpenAI-compatible JSON)
	var result Result
	if err := json.Unmarshal(respBody, &result); err != nil {
		return "", fmt.Errorf("failed to parse OpenRouter response: %v", err)
	}

	if len(result.Choices) > 0 {
		return result.Choices[0].Message.Content, nil
	}

	return "No response from AI agent", nil
}

// callFakeAgent simulates an agent response for testing
func callFakeAgent(ctx context.Context, agent *pb.Agent, prompt string, taskGoal string) (string, error) {
	// Simulate some processing time if needed
	// time.Sleep(10 * time.Millisecond)
	return fmt.Sprintf("Fake agent response for task: %s", taskGoal), nil
}
