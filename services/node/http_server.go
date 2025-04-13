package node

import (
	"encoding/json"
	"log"
	"net/http"
)

// MCPToolDiscoveryServer exposes the MCP tool registry over HTTP for agent harness discovery.
type MCPToolDiscoveryServer struct {
	Registry MCPToolRegistry
}

// NewMCPToolDiscoveryServer creates a new HTTP server for tool discovery.
func NewMCPToolDiscoveryServer(registry MCPToolRegistry) *MCPToolDiscoveryServer {
	return &MCPToolDiscoveryServer{Registry: registry}
}

// Start launches the HTTP server on the given address (e.g., ":8080").
func (s *MCPToolDiscoveryServer) Start(addr string) {
	http.HandleFunc("/mcp/tools", s.handleListTools)
	log.Printf("MCP Tool Discovery HTTP server listening on %s", addr)
	go func() {
		if err := http.ListenAndServe(addr, nil); err != nil {
			log.Fatalf("MCP Tool Discovery HTTP server failed: %v", err)
		}
	}()
}

// handleListTools returns the list of registered tools and their schemas as JSON.
func (s *MCPToolDiscoveryServer) handleListTools(w http.ResponseWriter, r *http.Request) {
	tools := s.Registry.ListTools()
	w.Header().Set("Content-Type", "application/json")
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	if err := enc.Encode(tools); err != nil {
		http.Error(w, "Failed to encode tools", http.StatusInternalServerError)
	}
}
