package node

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestInMemoryToolRegistry_DynamicRegistrationAndVersioning(t *testing.T) {
	reg := NewInMemoryToolRegistry(nil)

	tool := MCPTool{
		Name:         "test-tool",
		Version:      "v2.1.0",
		Description:  "A test tool",
		InputSchema:  map[string]string{"foo": "string"},
		OutputSchema: map[string]string{"bar": "int"},
	}
	// Register tool
	if err := reg.RegisterTool(tool); err != nil {
		t.Fatalf("RegisterTool failed: %v", err)
	}
	// ListTools should include the tool with correct version
	found := false
	for _, t := range reg.ListTools() {
		if t.Name == "test-tool" && t.Version == "v2.1.0" {
			found = true
		}
	}
	if !found {
		t.Error("Registered tool with version not found in ListTools")
	}
	// Unregister tool
	if err := reg.UnregisterTool("test-tool"); err != nil {
		t.Fatalf("UnregisterTool failed: %v", err)
	}
	if len(reg.ListTools()) != 0 {
		t.Error("Tool was not removed after UnregisterTool")
	}
}

func TestInMemoryToolRegistry_SchemaValidation(t *testing.T) {
	reg := NewInMemoryToolRegistry(nil)
	invalidTool := MCPTool{
		Name:         "bad-tool",
		Version:      "v1",
		Description:  "Bad tool",
		InputSchema:  map[string]string{"foo": "notatype"},
		OutputSchema: map[string]string{"bar": "int"},
	}
	if err := reg.RegisterTool(invalidTool); err == nil {
		t.Error("Expected error for invalid input schema type, got nil")
	}
}

func TestMCPToolDiscoveryServer_Endpoint(t *testing.T) {
	reg := NewInMemoryToolRegistry([]MCPTool{
		{
			Name:         "discovery-tool",
			Version:      "v1.2.3",
			Description:  "Discovery test tool",
			InputSchema:  map[string]string{"x": "string"},
			OutputSchema: map[string]string{"y": "bool"},
		},
	})
	server := NewMCPToolDiscoveryServer(reg)
	req := httptest.NewRequest("GET", "/mcp/tools", nil)
	w := httptest.NewRecorder()
	server.handleListTools(w, req)
	resp := w.Result()
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("Expected 200 OK, got %d", resp.StatusCode)
	}
	var tools []MCPTool
	if err := json.NewDecoder(resp.Body).Decode(&tools); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}
	if len(tools) != 1 || tools[0].Name != "discovery-tool" || tools[0].Version != "v1.2.3" {
		t.Errorf("Tool discovery endpoint returned wrong data: %+v", tools)
	}
}

// Adapter that always returns an error
type errorAdapter struct{}

func (a *errorAdapter) Invoke(ctx context.Context, tool MCPTool, input map[string]interface{}) (map[string]interface{}, error) {
	return nil, fmt.Errorf("adapter error: failed to invoke tool")
}

func TestInvokeMCPTool_AdapterError(t *testing.T) {
	reg := NewInMemoryToolRegistry([]MCPTool{
		{
			Name:         "error-tool",
			Version:      "v1",
			Description:  "Tool that always errors",
			InputSchema:  map[string]string{"foo": "string"},
			OutputSchema: map[string]string{"bar": "int"},
		},
	})
	adapterSelector := func(tool MCPTool) MCPToolAdapter { return &errorAdapter{} }
	input := map[string]interface{}{"foo": "bar"}
	_, err := InvokeMCPTool(context.Background(), "error-tool", input, reg, adapterSelector)
	if err == nil || err.Error() == "" || (!strings.Contains(err.Error(), "adapter error")) {
		t.Errorf("Expected adapter error to propagate, got: %v", err)
	}
}

// Adapter that simulates a timeout
type timeoutAdapter struct{}

func (a *timeoutAdapter) Invoke(ctx context.Context, tool MCPTool, input map[string]interface{}) (map[string]interface{}, error) {
	<-ctx.Done()
	return nil, ctx.Err()
}

func TestInvokeMCPTool_AdapterTimeout(t *testing.T) {
	reg := NewInMemoryToolRegistry([]MCPTool{
		{
			Name:         "timeout-tool",
			Version:      "v1",
			Description:  "Tool that times out",
			InputSchema:  map[string]string{"foo": "string"},
			OutputSchema: map[string]string{"bar": "int"},
		},
	})
	adapterSelector := func(tool MCPTool) MCPToolAdapter { return &timeoutAdapter{} }
	input := map[string]interface{}{"foo": "bar"}
	ctx, cancel := context.WithTimeout(context.Background(), 1)
	defer cancel()
	_, err := InvokeMCPTool(ctx, "timeout-tool", input, reg, adapterSelector)
	if err == nil || ctx.Err() == nil || err != ctx.Err() {
		t.Errorf("Expected context deadline exceeded error, got: %v", err)
	}
}

// Adapter that returns a partial result
type partialAdapter struct{}

func (a *partialAdapter) Invoke(ctx context.Context, tool MCPTool, input map[string]interface{}) (map[string]interface{}, error) {
	return map[string]interface{}{
		"status":  "PARTIAL",
		"summary": "Partial result",
		"output":  "",
	}, nil
}

func TestInvokeMCPTool_AdapterPartialResult(t *testing.T) {
	reg := NewInMemoryToolRegistry([]MCPTool{
		{
			Name:         "partial-tool",
			Version:      "v1",
			Description:  "Tool that returns partial result",
			InputSchema:  map[string]string{"foo": "string"},
			OutputSchema: map[string]string{"bar": "int"},
		},
	})
	adapterSelector := func(tool MCPTool) MCPToolAdapter { return &partialAdapter{} }
	input := map[string]interface{}{"foo": "bar"}
	result, err := InvokeMCPTool(context.Background(), "partial-tool", input, reg, adapterSelector)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if result["status"] != "PARTIAL" {
		t.Errorf("Expected status PARTIAL, got: %v", result["status"])
	}
	if result["summary"] != "Partial result" {
		t.Errorf("Expected summary 'Partial result', got: %v", result["summary"])
	}
}

func TestMCPToolToOpenAIFunctionSchema(t *testing.T) {
	tool := MCPTool{
		Name:        "schema-tool",
		Version:     "v1",
		Description: "Tool for schema conversion",
		InputSchema: map[string]string{
			"s":     "string",
			"i":     "int",
			"b":     "bool",
			"f":     "float",
			"alias": "integer",
		},
		OutputSchema: map[string]string{"out": "string"},
	}
	schema := MCPToolToOpenAIFunctionSchema(tool)
	if schema.Name != tool.Name {
		t.Errorf("Expected name %q, got %q", tool.Name, schema.Name)
	}
	if schema.Description != tool.Description {
		t.Errorf("Expected description %q, got %q", tool.Description, schema.Description)
	}
	props := schema.Parameters.Properties
	if props["s"].Type != "string" {
		t.Errorf("Expected 's' to be type string, got %q", props["s"].Type)
	}
	if props["i"].Type != "integer" {
		t.Errorf("Expected 'i' to be type integer, got %q", props["i"].Type)
	}
	if props["b"].Type != "boolean" {
		t.Errorf("Expected 'b' to be type boolean, got %q", props["b"].Type)
	}
	if props["f"].Type != "number" {
		t.Errorf("Expected 'f' to be type number, got %q", props["f"].Type)
	}
	if props["alias"].Type != "integer" {
		t.Errorf("Expected 'alias' to be type integer, got %q", props["alias"].Type)
	}
	// All fields should be required
	for _, param := range []string{"s", "i", "b", "f", "alias"} {
		found := false
		for _, req := range schema.Parameters.Required {
			if req == param {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected %q to be required", param)
		}
	}
}
