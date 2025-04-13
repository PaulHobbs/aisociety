package node

import (
	"context"
	"fmt"
	"sync"
)

// MCPTool describes a tool's metadata and schema.
type MCPTool struct {
	Name         string
	Version      string
	Description  string
	InputSchema  map[string]string // For simplicity, just a map of param name to type
	OutputSchema map[string]string
}

// MCPToolRegistry provides discovery of available MCP tools.
type MCPToolRegistry interface {
	ListTools() []MCPTool
	GetTool(name string) (*MCPTool, error)
	RegisterTool(tool MCPTool) error
	UnregisterTool(name string) error
}

// InMemoryToolRegistry is a simple registry for testing/demo.

type InMemoryToolRegistry struct {
	tools map[string]MCPTool
	mu    sync.RWMutex
}

func NewInMemoryToolRegistry(tools []MCPTool) *InMemoryToolRegistry {
	m := make(map[string]MCPTool)
	for _, t := range tools {
		m[t.Name] = t
	}
	return &InMemoryToolRegistry{tools: m}
}

func (r *InMemoryToolRegistry) ListTools() []MCPTool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var out []MCPTool
	for _, t := range r.tools {
		out = append(out, t)
	}
	return out
}

func (r *InMemoryToolRegistry) GetTool(name string) (*MCPTool, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	t, ok := r.tools[name]
	if !ok {
		return nil, fmt.Errorf("tool not found: %s", name)
	}
	return &t, nil
}

// ValidateInputSchema checks that the input matches the tool's input schema.
func ValidateInputSchema(tool MCPTool, input map[string]interface{}) error {
	validTypes := map[string]func(interface{}) bool{
		"string": func(v interface{}) bool { _, ok := v.(string); return ok },
		"int":    func(v interface{}) bool { _, ok := v.(int); return ok },
		"bool":   func(v interface{}) bool { _, ok := v.(bool); return ok },
		"float":  func(v interface{}) bool { _, ok := v.(float64); return ok },
	}
	for param, typ := range tool.InputSchema {
		val, ok := input[param]
		if !ok {
			return fmt.Errorf("missing required input parameter: %s", param)
		}
		validator, ok := validTypes[typ]
		if !ok {
			return fmt.Errorf("unsupported input type in schema: %s", typ)
		}
		if !validator(val) {
			return fmt.Errorf("input parameter '%s' has wrong type: expected %s", param, typ)
		}
	}
	return nil
}

// ToolAdapterSelector returns the adapter for a given tool.
// For now, always returns the provided defaultAdapter.
type ToolAdapterSelector func(tool MCPTool) MCPToolAdapter

// InvokeMCPTool validates input, selects adapter, and invokes the tool.
func InvokeMCPTool(
	ctx context.Context,
	toolName string,
	input map[string]interface{},
	registry MCPToolRegistry,
	adapterSelector ToolAdapterSelector,
) (map[string]interface{}, error) {
	tool, err := registry.GetTool(toolName)
	if err != nil {
		return nil, fmt.Errorf("tool not found: %w", err)
	}
	if err := ValidateInputSchema(*tool, input); err != nil {
		return nil, fmt.Errorf("input schema validation failed: %w", err)
	}
	adapter := adapterSelector(*tool)
	if adapter == nil {
		return nil, fmt.Errorf("no adapter available for tool: %s", toolName)
	}
	return adapter.Invoke(ctx, *tool, input)
}

// RegisterTool adds or updates a tool in the registry, validating its schema and version.
func (r *InMemoryToolRegistry) RegisterTool(tool MCPTool) error {
	if tool.Name == "" {
		return fmt.Errorf("tool name is required")
	}
	if tool.Version == "" {
		return fmt.Errorf("tool version is required")
	}
	// Basic schema validation: ensure input/output schemas are not nil
	if tool.InputSchema == nil {
		return fmt.Errorf("input schema is required")
	}
	if tool.OutputSchema == nil {
		return fmt.Errorf("output schema is required")
	}
	// Validate types in schemas (simple check: only allow string, int, bool, float)
	validTypes := map[string]bool{"string": true, "int": true, "bool": true, "float": true}
	for k, typ := range tool.InputSchema {
		if !validTypes[typ] {
			return fmt.Errorf("unsupported input type '%s' for key '%s'", typ, k)
		}
	}
	for k, typ := range tool.OutputSchema {
		if !validTypes[typ] {
			return fmt.Errorf("unsupported output type '%s' for key '%s'", typ, k)
		}
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.tools[tool.Name] = tool
	return nil
}

// UnregisterTool removes a tool from the registry.
func (r *InMemoryToolRegistry) UnregisterTool(name string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.tools[name]; !ok {
		return fmt.Errorf("tool not found: %s", name)
	}
	delete(r.tools, name)
	return nil
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
