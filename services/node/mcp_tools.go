package node

import (
	"context"
	"fmt"
)

// MCPTool describes a tool's metadata and schema.
type MCPTool struct {
	Name         string
	Description  string
	InputSchema  map[string]string // For simplicity, just a map of param name to type
	OutputSchema map[string]string
}

// MCPToolRegistry provides discovery of available MCP tools.
type MCPToolRegistry interface {
	ListTools() []MCPTool
	GetTool(name string) (*MCPTool, error)
}

// InMemoryToolRegistry is a simple registry for testing/demo.
type InMemoryToolRegistry struct {
	tools map[string]MCPTool
}

func NewInMemoryToolRegistry(tools []MCPTool) *InMemoryToolRegistry {
	m := make(map[string]MCPTool)
	for _, t := range tools {
		m[t.Name] = t
	}
	return &InMemoryToolRegistry{tools: m}
}

func (r *InMemoryToolRegistry) ListTools() []MCPTool {
	var out []MCPTool
	for _, t := range r.tools {
		out = append(out, t)
	}
	return out
}

func (r *InMemoryToolRegistry) GetTool(name string) (*MCPTool, error) {
	t, ok := r.tools[name]
	if !ok {
		return nil, fmt.Errorf("tool not found: %s", name)
	}
	return &t, nil
}

// MCPToolAdapter invokes a tool and returns the result.
type MCPToolAdapter interface {
	Invoke(ctx context.Context, tool MCPTool, input map[string]interface{}) (map[string]interface{}, error)
}

// MockToolAdapter is a mock implementation for testing.
type MockToolAdapter struct{}

func (a *MockToolAdapter) Invoke(ctx context.Context, tool MCPTool, input map[string]interface{}) (map[string]interface{}, error) {
	// For testing, just echo the input and tool name.
	return map[string]interface{}{
		"status":  "PASS",
		"summary": fmt.Sprintf("Invoked tool %s with input %v", tool.Name, input),
		"output":  fmt.Sprintf("Tool %s executed successfully", tool.Name),
	}, nil
}
