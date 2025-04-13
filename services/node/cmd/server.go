package main

import (
	"fmt"
	"net"
	"os"

	pb "paul.hobbs.page/aisociety/protos"
	"paul.hobbs.page/aisociety/services/node"

	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "50051"
	}
	addr := ":" + port

	// --- MCP Tool Registry Initialization ---
	toolRegistry := node.NewInMemoryToolRegistry([]node.MCPTool{
		{
			Name:         "knowledge-base-curator.summarize",
			Version:      "v1.0.0",
			Description:  "Summarizes knowledge base entries",
			InputSchema:  map[string]string{"text": "string"},
			OutputSchema: map[string]string{"summary": "string"},
		},
		{
			Name:         "hypothetical-tool.do-something",
			Version:      "v1.0.0",
			Description:  "A tool that requires text input",
			InputSchema:  map[string]string{"text": "string"},
			OutputSchema: map[string]string{"result": "string"},
		},
	})

	// --- Start MCP Tool Discovery HTTP Server ---
	httpDiscovery := node.NewMCPToolDiscoveryServer(toolRegistry)
	httpDiscovery.Start(":8080") // Listen on port 8080 for tool discovery

	lis, err := net.Listen("tcp", addr)
	if err != nil {
		panic(fmt.Sprintf("failed to listen on %s: %v", addr, err))
	}

	// Pass the registry to the NodeService
	s := grpc.NewServer()
	pb.RegisterNodeServiceServer(s, node.NewServerWithRegistry(toolRegistry))

	// Enable reflection for debugging and testing (optional but recommended)
	reflection.Register(s)

	fmt.Printf("NodeService server listening on port %s (gRPC), tool discovery on 8080\n", port)
	if err := s.Serve(lis); err != nil {
		panic(fmt.Sprintf("failed to serve: %v", err))
	}
}
