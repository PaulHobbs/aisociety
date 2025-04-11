package api

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
	pb "paul.hobbs.page/aisociety/protos"
	"paul.hobbs.page/aisociety/services/workflow/persistence"
)

// WorkflowServiceServerImpl implements pb.WorkflowServiceServer
type WorkflowServiceServerImpl struct {
	pb.UnimplementedWorkflowServiceServer
	StateManager persistence.StateManager
	EventLogger  EventLogger
}

func NewWorkflowServiceServer(sm persistence.StateManager, logger EventLogger) *WorkflowServiceServerImpl {
	return &WorkflowServiceServerImpl{
		StateManager: sm,
		EventLogger:  logger,
	}
}

func (s *WorkflowServiceServerImpl) CreateWorkflow(ctx context.Context, req *pb.CreateWorkflowRequest) (*pb.CreateWorkflowResponse, error) {
	// Generate a new UUID for the workflow
	workflowID := uuid.New().String()

	// Prepare workflow struct
	workflow := &persistence.Workflow{
		ID:    workflowID,
		Nodes: req.GetNodes(),
		// Optionally set Name, Description, Status if available in request
	}

	// Persist workflow metadata
	returnedID, err := s.StateManager.CreateWorkflow(ctx, workflow)
	if err != nil {
		return nil, err
	}
	workflowID = returnedID

	// Persist initial nodes
	for _, node := range req.GetNodes() {
		err := s.StateManager.CreateNode(ctx, workflowID, node)
		if err != nil {
			return nil, err
		}
	}

	// Emit WorkflowCreated event
	if s.EventLogger != nil {
		payloadBytes, err := proto.Marshal(req)
		if err == nil {
			event := Event{
				Type:      EventWorkflowCreated,
				Timestamp: time.Now(),
				ProtoType: "CreateWorkflowRequest",
				Payload:   payloadBytes,
			}
			s.EventLogger.LogEvent(event)
		}
	}

	// Return response with workflow ID
	return &pb.CreateWorkflowResponse{
		WorkflowId: workflowID,
	}, nil
}

func (s *WorkflowServiceServerImpl) GetWorkflow(ctx context.Context, req *pb.GetWorkflowRequest) (*pb.GetWorkflowResponse, error) {
	workflowID := req.GetWorkflowId()
	if workflowID == "" {
		return nil, status.Errorf(codes.InvalidArgument, "workflow_id is required")
	}

	workflow, err := s.StateManager.GetWorkflow(ctx, workflowID)
	if err != nil {
		// Check for not found error
		if err == persistence.ErrWorkflowNotFound {
			return nil, status.Errorf(codes.NotFound, "workflow %s not found", workflowID)
		}
		// Other DB errors
		return nil, status.Errorf(codes.Internal, "failed to get workflow: %v", err)
	}

	if workflow == nil {
		return nil, status.Errorf(codes.NotFound, "workflow %s not found", workflowID)
	}

	resp := &pb.GetWorkflowResponse{
		Nodes: workflow.Nodes,
	}

	return resp, nil
}

func (s *WorkflowServiceServerImpl) ListWorkflows(ctx context.Context, req *pb.ListWorkflowsRequest) (*pb.ListWorkflowsResponse, error) {
	ids, err := s.StateManager.ListWorkflows(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to list workflows: %v", err)
	}
	return &pb.ListWorkflowsResponse{
		WorkflowIds: ids,
	}, nil
}

func (s *WorkflowServiceServerImpl) UpdateWorkflow(ctx context.Context, req *pb.UpdateWorkflowRequest) (*pb.UpdateWorkflowResponse, error) {
	workflowID := req.GetWorkflowId()
	if workflowID == "" {
		return nil, status.Errorf(codes.InvalidArgument, "workflow_id is required")
	}

	// Fetch current workflow
	workflow, err := s.StateManager.GetWorkflow(ctx, workflowID)
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "workflow %s not found", workflowID)
	}

	// Build maps of current and incoming nodes
	currentNodes := make(map[string]*pb.Node)
	for _, n := range workflow.Nodes {
		currentNodes[n.GetNodeId()] = n
	}

	incomingNodes := make(map[string]*pb.Node)
	for _, n := range req.GetNodes() {
		// Generate ID if missing (insert case)
		if n.GetNodeId() == "" {
			n.NodeId = uuid.New().String()
		}
		incomingNodes[n.GetNodeId()] = n
	}

	var edits []*pb.NodeEdit
	now := time.Now()

	// Inserts and updates
	for id, newNode := range incomingNodes {
		oldNode, exists := currentNodes[id]
		if !exists {
			// Insert
			edits = append(edits, &pb.NodeEdit{
				Type:        pb.NodeEdit_INSERT,
				Timestamp:   timestamppb.New(now),
				Description: "Insert node",
				Node:        newNode,
			})
		} else {
			// Check if node differs (simplified: compare serialized bytes)
			if !proto.Equal(oldNode, newNode) {
				edits = append(edits, &pb.NodeEdit{
					Type:        pb.NodeEdit_UPDATE,
					Timestamp:   timestamppb.New(now),
					Description: "Update node",
					Node:        newNode,
				})
			}
		}
	}

	// Deletes
	for id, oldNode := range currentNodes {
		if _, exists := incomingNodes[id]; !exists {
			edits = append(edits, &pb.NodeEdit{
				Type:        pb.NodeEdit_DELETE,
				Timestamp:   timestamppb.New(now),
				Description: "Delete node",
				Node:        oldNode,
			})
		}
	}

	// Apply edits transactionally
	if len(edits) > 0 {
		if err := s.StateManager.ApplyNodeEdits(ctx, workflowID, edits); err != nil {
			return &pb.UpdateWorkflowResponse{Success: false}, status.Errorf(codes.Internal, "failed to apply node edits: %v", err)
		}
	}

	// Emit WorkflowUpdated event
	if s.EventLogger != nil {
		payloadBytes, err := proto.Marshal(req)
		if err == nil {
			event := Event{
				Type:      EventWorkflowUpdated,
				Timestamp: time.Now(),
				ProtoType: "UpdateWorkflowRequest",
				Payload:   payloadBytes,
			}
			s.EventLogger.LogEvent(event)
		}
	}

	return &pb.UpdateWorkflowResponse{Success: true}, nil
}

func (s *WorkflowServiceServerImpl) GetNode(ctx context.Context, req *pb.GetNodeRequest) (*pb.GetNodeResponse, error) {
	workflowID := req.GetWorkflowId()
	nodeID := req.GetNodeId()

	node, err := s.StateManager.GetNode(ctx, workflowID, nodeID)
	if err != nil {
		// Check for not found error if StateManager implementation exposes it
		if errors.Is(err, sql.ErrNoRows) {
			return nil, status.Errorf(codes.NotFound, "node %s not found in workflow %s", nodeID, workflowID)
		}
		// Generic internal error
		return nil, status.Errorf(codes.Internal, "failed to get node: %v", err)
	}
	if node == nil {
		return nil, status.Errorf(codes.NotFound, "node %s not found in workflow %s", nodeID, workflowID)
	}

	return &pb.GetNodeResponse{
		Node: node,
	}, nil
}

func (s *WorkflowServiceServerImpl) UpdateNode(ctx context.Context, req *pb.UpdateNodeRequest) (*pb.UpdateNodeResponse, error) {
	err := s.StateManager.UpdateNode(ctx, req.WorkflowId, req.Node)
	if err != nil {
		if err == persistence.ErrWorkflowNotFound {
			return &pb.UpdateNodeResponse{Success: false}, status.Errorf(codes.NotFound, "workflow not found: %v", err)
		}
		return &pb.UpdateNodeResponse{Success: false}, status.Errorf(codes.Internal, "failed to update node: %v", err)
	}

	// Emit event based on node status
	if s.EventLogger != nil && req.Node != nil {
		payloadBytes, err := proto.Marshal(req)
		if err == nil {
			eventType := EventNodeUpdated
			switch req.Node.Status {
			case 10: // RUNNING
				eventType = EventNodeDispatched
			case 1, 2, 3, 4, 5, 6, 7, 8: // PASS, FAIL, SKIPPED, FILTERED, TASK_ERROR, INFRA_ERROR, TIMEOUT, CRASH
				eventType = EventNodeCompleted
			}
			event := Event{
				Type:      eventType,
				Timestamp: time.Now(),
				ProtoType: "UpdateNodeRequest",
				Payload:   payloadBytes,
			}
			s.EventLogger.LogEvent(event)
		}
	}
	return &pb.UpdateNodeResponse{Success: true}, nil
}
