package persistence

import (
	"context"
	"errors"

	pb "paul.hobbs.page/aisociety/protos"
)

var ErrWorkflowNotFound = errors.New("workflow not found")

// StateManager defines the interface for workflow state persistence operations.
// It abstracts the database operations for storing and retrieving workflow data.
type StateManager interface {
	// Workflow operations
	CreateWorkflow(ctx context.Context, workflow *Workflow) (string, error)
	GetWorkflow(ctx context.Context, workflowID string) (*Workflow, error)
	UpdateWorkflowStatus(ctx context.Context, workflowID string, status pb.Status) error
	ListWorkflows(ctx context.Context) ([]string, error)

	// Node operations
	CreateNode(ctx context.Context, workflowID string, node *pb.Node) error
	GetNode(ctx context.Context, workflowID, nodeID string) (*pb.Node, error)
	UpdateNode(ctx context.Context, workflowID string, node *pb.Node) error
	ApplyNodeEdits(ctx context.Context, workflowID string, edits []*pb.NodeEdit) error

	// Query operations
	FindReadyNodes(ctx context.Context) ([]*pb.Node, error)

	// Close the state manager and release resources
	Close() error
}

// Workflow represents a workflow entity as stored in the database
type Workflow struct {
	ID          string
	Name        string
	Description string
	Status      pb.Status
	Nodes       []*pb.Node // In-memory representation of nodes
}
