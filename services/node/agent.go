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

// OpenAIFunctionParamSchema represents a JSON Schema property for a function parameter.
type OpenAIFunctionParamSchema struct {
	Type        string `json:"type"`
	Description string `json:"description,omitempty"`
}

// OpenAIFunctionSchema represents the OpenAI function/tool schema.
type OpenAIFunctionSchema struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Parameters  struct {
		Type       string                               `json:"type"`
		Properties map[string]OpenAIFunctionParamSchema `json:"properties"`
		Required   []string                             `json:"required,omitempty"`
	} `json:"parameters"`
}

// Convert MCPTool to OpenAIFunctionSchema (basic mapping).
func MCPToolToOpenAIFunctionSchema(tool MCPTool) OpenAIFunctionSchema {
	schema := OpenAIFunctionSchema{
		Name:        tool.Name,
		Description: tool.Description,
	}
	schema.Parameters.Type = "object"
	schema.Parameters.Properties = map[string]OpenAIFunctionParamSchema{}
	for param, typ := range tool.InputSchema {
		// Map simple types to JSON Schema types (string, integer, boolean, etc.)
		jsonType := "string"
		switch typ {
		case "int", "integer":
			jsonType = "integer"
		case "bool", "boolean":
			jsonType = "boolean"
		case "number", "float", "double":
			jsonType = "number"
		}
		schema.Parameters.Properties[param] = OpenAIFunctionParamSchema{
			Type: jsonType,
		}
		schema.Parameters.Required = append(schema.Parameters.Required, param)
	}
	return schema
}

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
/*
callRealAgent handles the interaction with the actual OpenRouter API.

This version integrates the MCP tool registry to surface available tools to the agent.
If the OpenRouter API supports OpenAI function/tool calling, available tools are passed in the "tools" field.
If not supported, see documentation below for a proposed workaround.

NOTE: Tool invocation/result handling is NOT implemented here—this only surfaces tool options to the agent.
*/
func callRealAgent(ctx context.Context, agent *pb.Agent, prompt string, toolRegistry MCPToolRegistry) (string, error) {
	apiURL := "https://openrouter.ai/api/v1/chat/completions"
	apiKey := os.Getenv("OPENROUTER_API_KEY")
	if apiKey == "" {
		return "", fmt.Errorf("missing OpenRouter API key")
	}

	// Discover available tools from the registry
	var toolsPayload []OpenAIFunctionSchema
	if toolRegistry != nil {
		for _, tool := range toolRegistry.ListTools() {
			toolsPayload = append(toolsPayload, MCPToolToOpenAIFunctionSchema(tool))
		}
	}

	// Build the payload for OpenAI/OpenRouter API
	payloadMap := map[string]interface{}{
		"model": agent.GetModelType(),
		"messages": []map[string]string{
			{"role": "user", "content": prompt},
		},
	}
	// If tools are available, add them to the payload (OpenAI function calling)
	if len(toolsPayload) > 0 {
		payloadMap["tools"] = toolsPayload
	}

	payloadBytes, err := json.Marshal(payloadMap)
	if err != nil {
		return "", fmt.Errorf("failed to marshal payload: %v", err)
	}

	reqBody := strings.NewReader(string(payloadBytes))
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

/*
==== Documentation: Tool Surfacing for LLM-based Agents ====

- If the OpenAI/OpenRouter API supports function/tool calling (i.e., accepts a "tools" field in the chat/completions payload), available tools and their schemas are passed as shown above.
- If the API does NOT support passing tool schemas directly:
    - This is a gap for LLM-based agents, as they will not be aware of available tools or their invocation schema.
    - Proposed solution: Surface tool options by injecting a description of available tools and their parameters into the system prompt or user message. This allows the LLM to "see" tool options, though it cannot invoke them natively.
    - For advanced use cases, a custom function-calling interface could be implemented, where the LLM is instructed to output tool calls in a specific format, which the harness can parse and route.

- This implementation does NOT handle tool invocation or result handling—only surfacing tool options at invocation time.

*/

// callFakeAgent simulates an agent response for testing
func callFakeAgent(ctx context.Context, agent *pb.Agent, prompt string, taskGoal string) (string, error) {
	// Simulate some processing time if needed
	// time.Sleep(10 * time.Millisecond)
	return fmt.Sprintf("Fake agent response for task: %s", taskGoal), nil
}
