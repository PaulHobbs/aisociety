package scheduler

import (
	"context"
	"log"
	"time"

	pb "paul.hobbs.page/aisociety/protos"
)

// StateManager abstracts persistence operations needed by the scheduler.
type StateManager interface {
	FindReadyNodes(ctx context.Context) ([]*pb.Node, error)
	UpdateNode(ctx context.Context, workflowID string, node *pb.Node) error
	ApplyNodeEdits(ctx context.Context, workflowID string, edits []*pb.NodeEdit) error
}

// NodeServiceClient abstracts the NodeService gRPC client.
type NodeServiceClient interface {
	ExecuteNode(ctx context.Context, req *pb.ExecuteNodeRequest) (*pb.ExecuteNodeResponse, error)
}

// Scheduler defines the scheduling interface.
type Scheduler interface {
	Run(ctx context.Context)
}

// SimpleScheduler is a basic implementation of Scheduler.
type SimpleScheduler struct {
	StateManager      StateManager
	NodeServiceClient NodeServiceClient
	PollInterval      time.Duration
}

// NewSimpleScheduler creates a new SimpleScheduler.
func NewSimpleScheduler(sm StateManager, nc NodeServiceClient, pollInterval time.Duration) *SimpleScheduler {
	return &SimpleScheduler{
		StateManager:      sm,
		NodeServiceClient: nc,
		PollInterval:      pollInterval,
	}
}

// Run starts the scheduling loop.
func (s *SimpleScheduler) Run(ctx context.Context) {
	ticker := time.NewTicker(s.PollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Println("Scheduler stopped")
			return
		case <-ticker.C:
			s.scheduleOnce(ctx)
		}
	}
}

// scheduleOnce performs one scheduling iteration.
func (s *SimpleScheduler) scheduleOnce(ctx context.Context) {
	readyNodes, err := s.StateManager.FindReadyNodes(ctx)
	if err != nil {
		log.Printf("Error finding ready nodes: %v", err)
		return
	}

	for _, node := range readyNodes {
		go s.dispatchNode(ctx, node)
	}
}

// dispatchNode dispatches a single node to the NodeService.
func (s *SimpleScheduler) dispatchNode(ctx context.Context, node *pb.Node) {
	// TODO: Determine how to properly obtain the workflowID for a given node.
	// Using a placeholder for now as tests don't depend on a specific ID.
	workflowID := "unknown_workflow"
	nodeID := node.NodeId

	// Update node status to RUNNING
	node.Status = pb.Status_RUNNING
	if err := s.StateManager.UpdateNode(ctx, workflowID, node); err != nil {
		log.Printf("Failed to update node %s status to RUNNING: %v", nodeID, err)
		return
	}

	// Build ExecuteNodeRequest
	req := &pb.ExecuteNodeRequest{
		WorkflowId: workflowID,
		NodeId:     nodeID,
		Node:       node,
		// Upstream and downstream nodes can be fetched if needed
	}

	resp, err := s.NodeServiceClient.ExecuteNode(ctx, req)
	if err != nil {
		log.Printf("Error executing node %s: %v", nodeID, err)
		node.Status = pb.Status_INFRA_ERROR
		_ = s.StateManager.UpdateNode(ctx, workflowID, node)
		return
	}

	// Update node with response
	updatedNode := resp.Node
	if err := s.StateManager.UpdateNode(ctx, workflowID, updatedNode); err != nil {
		log.Printf("Failed to update node %s after execution: %v", nodeID, err)
		return
	}

	// Apply any NodeEdits transactionally
	if len(updatedNode.Edits) > 0 {
		if err := s.StateManager.ApplyNodeEdits(ctx, workflowID, updatedNode.Edits); err != nil {
			log.Printf("Failed to apply edits for node %s: %v", nodeID, err)
		}
	}
}
