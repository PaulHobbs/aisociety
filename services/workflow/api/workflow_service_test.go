package api

import (
	"context"
	"errors"
	"testing"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	pb "paul.hobbs.page/aisociety/protos"
	"paul.hobbs.page/aisociety/services/workflow/persistence"
)

// fakeStateManager is a fake implementation of persistence.StateManager
type fakeStateManager struct {
	CreateWorkflowFunc func(ctx context.Context, workflow *persistence.Workflow) (string, error)
	GetWorkflowFunc    func(ctx context.Context, workflowID string) (*persistence.Workflow, error)
	ListWorkflowsFunc  func(ctx context.Context) ([]string, error)
	ApplyNodeEditsFunc func(ctx context.Context, workflowID string, edits []*pb.NodeEdit) error
	GetNodeFunc        func(ctx context.Context, workflowID, nodeID string) (*pb.Node, error)
	UpdateNodeFunc     func(ctx context.Context, workflowID string, node *pb.Node) error
}

func (m *fakeStateManager) CreateWorkflow(ctx context.Context, workflow *persistence.Workflow) (string, error) {
	if m.CreateWorkflowFunc != nil {
		return m.CreateWorkflowFunc(ctx, workflow)
	}
	return "fake-id", nil
}

func (m *fakeStateManager) GetWorkflow(ctx context.Context, workflowID string) (*persistence.Workflow, error) {
	if m.GetWorkflowFunc != nil {
		return m.GetWorkflowFunc(ctx, workflowID)
	}
	return nil, nil
}

func (m *fakeStateManager) CreateNode(ctx context.Context, workflowID string, node *pb.Node) error {
	return nil
}
func (m *fakeStateManager) ApplyNodeEdits(ctx context.Context, workflowID string, edits []*pb.NodeEdit) error {
	if m.ApplyNodeEditsFunc != nil {
		return m.ApplyNodeEditsFunc(ctx, workflowID, edits)
	}
	return nil
}

func (m *fakeStateManager) UpdateNode(ctx context.Context, workflowID string, node *pb.Node) error {
	if m.UpdateNodeFunc != nil {
		return m.UpdateNodeFunc(ctx, workflowID, node)
	}
	return nil
}

func (m *fakeStateManager) GetNode(ctx context.Context, workflowID, nodeID string) (*pb.Node, error) {
	return m.GetNodeFunc(ctx, workflowID, nodeID)
}
func (m *fakeStateManager) ListWorkflows(ctx context.Context) ([]string, error) {
	if m.ListWorkflowsFunc != nil {
		return m.ListWorkflowsFunc(ctx)
	}
	return nil, nil
}
func (m *fakeStateManager) Close() error {
	return nil
}
func (m *fakeStateManager) FindReadyNodes(ctx context.Context) ([]*pb.Node, error) {
	return nil, nil
}

func TestCreateWorkflow_Success(t *testing.T) {
	fakeSM := &fakeStateManager{
		CreateWorkflowFunc: func(ctx context.Context, workflow *persistence.Workflow) (string, error) {
			return "generated-id", nil
		},
	}

	server := NewWorkflowServiceServer(fakeSM, &StdoutEventLogger{})

	resp, err := server.CreateWorkflow(context.Background(), &pb.CreateWorkflowRequest{})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if resp.WorkflowId != "generated-id" {
		t.Errorf("expected workflow_id 'generated-id', got %s", resp.WorkflowId)
	}
}

func TestCreateWorkflow_Error(t *testing.T) {
	fakeSM := &fakeStateManager{
		CreateWorkflowFunc: func(ctx context.Context, workflow *persistence.Workflow) (string, error) {
			return "", errors.New("db error")
		},
	}
	server := NewWorkflowServiceServer(fakeSM, &StdoutEventLogger{})

	_, err := server.CreateWorkflow(context.Background(), &pb.CreateWorkflowRequest{})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestGetWorkflow(t *testing.T) {
	fakeSM := &fakeStateManager{}
	server := NewWorkflowServiceServer(fakeSM, &StdoutEventLogger{})

	t.Run("success", func(t *testing.T) {
		fakeSM.GetWorkflowFunc = func(ctx context.Context, workflowID string) (*persistence.Workflow, error) {
			return &persistence.Workflow{
				ID: workflowID,
				Nodes: []*pb.Node{
					{NodeId: "node1", Description: "Node 1"},
					{NodeId: "node2", Description: "Node 2"},
				},
			}, nil
		}

		resp, err := server.GetWorkflow(context.Background(), &pb.GetWorkflowRequest{WorkflowId: "wf-123"})
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if len(resp.Nodes) != 2 {
			t.Errorf("expected 2 nodes, got %d", len(resp.Nodes))
		}
	})

	t.Run("not found", func(t *testing.T) {
		fakeSM.GetWorkflowFunc = func(ctx context.Context, workflowID string) (*persistence.Workflow, error) {
			return nil, persistence.ErrWorkflowNotFound
		}

		_, err := server.GetWorkflow(context.Background(), &pb.GetWorkflowRequest{WorkflowId: "missing-wf"})
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		st, ok := status.FromError(err)
		if !ok || st.Code() != codes.NotFound {
			t.Fatalf("expected NotFound error, got %v", err)
		}
	})

	t.Run("db error", func(t *testing.T) {
		fakeSM.GetWorkflowFunc = func(ctx context.Context, workflowID string) (*persistence.Workflow, error) {
			return nil, errors.New("db failure")
		}

		_, err := server.GetWorkflow(context.Background(), &pb.GetWorkflowRequest{WorkflowId: "wf-err"})
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		st, ok := status.FromError(err)
		if !ok || st.Code() != codes.Internal {
			t.Fatalf("expected Internal error, got %v", err)
		}
	})
}

func TestListWorkflows_Success(t *testing.T) {
	fakeSM := &fakeStateManager{
		ListWorkflowsFunc: func(ctx context.Context) ([]string, error) {
			return []string{"wf1", "wf2"}, nil
		},
	}
	svc := &WorkflowServiceServerImpl{StateManager: fakeSM}

	resp, err := svc.ListWorkflows(context.Background(), &pb.ListWorkflowsRequest{})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(resp.WorkflowIds) != 2 || resp.WorkflowIds[0] != "wf1" || resp.WorkflowIds[1] != "wf2" {
		t.Errorf("unexpected workflow IDs: %v", resp.WorkflowIds)
	}
}

func TestListWorkflows_Error(t *testing.T) {
	fakeSM := &fakeStateManager{
		ListWorkflowsFunc: func(ctx context.Context) ([]string, error) {
			return nil, errors.New("db error")
		},
	}
	svc := &WorkflowServiceServerImpl{StateManager: fakeSM}

	_, err := svc.ListWorkflows(context.Background(), &pb.ListWorkflowsRequest{})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}
func TestUpdateWorkflow(t *testing.T) {
	fakeSM := &fakeStateManager{}
	server := NewWorkflowServiceServer(fakeSM, &StdoutEventLogger{})

	t.Run("success", func(t *testing.T) {
		fakeSM.GetWorkflowFunc = func(ctx context.Context, workflowID string) (*persistence.Workflow, error) {
			return &persistence.Workflow{
				ID: workflowID,
				Nodes: []*pb.Node{
					{NodeId: "node1", Description: "Old Node"},
				},
			}, nil
		}
		fakeSM.ApplyNodeEditsFunc = func(ctx context.Context, workflowID string, edits []*pb.NodeEdit) error {
			if len(edits) == 0 {
				t.Errorf("expected edits, got none")
			}
			return nil
		}

		req := &pb.UpdateWorkflowRequest{
			WorkflowId: "wf-123",
			Nodes: []*pb.Node{
				{NodeId: "node1", Description: "Updated Node"},
				{NodeId: "", Description: "New Node"},
			},
		}
		resp, err := server.UpdateWorkflow(context.Background(), req)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if !resp.Success {
			t.Errorf("expected success true, got false")
		}
	})

	t.Run("workflow not found", func(t *testing.T) {
		fakeSM.GetWorkflowFunc = func(ctx context.Context, workflowID string) (*persistence.Workflow, error) {
			return nil, errors.New("not found")
		}

		req := &pb.UpdateWorkflowRequest{
			WorkflowId: "missing-wf",
			Nodes:      []*pb.Node{},
		}
		_, err := server.UpdateWorkflow(context.Background(), req)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		st, _ := status.FromError(err)
		if st.Code() != codes.NotFound {
			t.Fatalf("expected NotFound error, got %v", err)
		}
	})

	t.Run("apply edits error", func(t *testing.T) {
		fakeSM.GetWorkflowFunc = func(ctx context.Context, workflowID string) (*persistence.Workflow, error) {
			return &persistence.Workflow{
				ID:    workflowID,
				Nodes: []*pb.Node{},
			}, nil
		}
		fakeSM.ApplyNodeEditsFunc = func(ctx context.Context, workflowID string, edits []*pb.NodeEdit) error {
			return errors.New("db failure")
		}

		req := &pb.UpdateWorkflowRequest{
			WorkflowId: "wf-123",
			Nodes: []*pb.Node{
				{NodeId: "node1", Description: "Node"},
			},
		}
		resp, err := server.UpdateWorkflow(context.Background(), req)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if resp.Success {
			t.Errorf("expected success false, got true")
		}
		st, _ := status.FromError(err)
		if st.Code() != codes.Internal {
			t.Fatalf("expected Internal error, got %v", err)
		}
	})
}
func TestGetNode(t *testing.T) {
	mockNode := &pb.Node{
		NodeId: "node-123",
		// Add other fields as needed
	}

	tests := []struct {
		name         string
		getNodeFunc  func(ctx context.Context, workflowID, nodeID string) (*pb.Node, error)
		expectedNode *pb.Node
		expectedCode codes.Code
	}{
		{
			name: "success",
			getNodeFunc: func(ctx context.Context, workflowID, nodeID string) (*pb.Node, error) {
				return mockNode, nil
			},
			expectedNode: mockNode,
			expectedCode: codes.OK,
		},
		{
			name: "not found",
			getNodeFunc: func(ctx context.Context, workflowID, nodeID string) (*pb.Node, error) {
				return nil, nil
			},
			expectedNode: nil,
			expectedCode: codes.NotFound,
		},
		// (removed invalid empty struct entry)
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			sm := &fakeStateManager{
				GetNodeFunc: tc.getNodeFunc,
			}
			svc := &WorkflowServiceServerImpl{
				StateManager: sm,
			}

			resp, err := svc.GetNode(context.Background(), &pb.GetNodeRequest{
				WorkflowId: "wf-1",
				NodeId:     "node-123",
			})

			if tc.expectedCode == codes.OK {
				if err != nil {
					t.Fatalf("expected no error, got %v", err)
				}
				if resp == nil || resp.Node == nil || resp.Node.GetNodeId() != tc.expectedNode.GetNodeId() {
					t.Fatalf("expected node %v, got %v", tc.expectedNode, resp)
				}
			} else {
				if err == nil {
					t.Fatalf("expected error code %v, got nil", tc.expectedCode)
				}
				st, ok := status.FromError(err)
				if !ok {
					t.Fatalf("expected gRPC status error, got %v", err)
				}
				if st.Code() != tc.expectedCode {
					t.Fatalf("expected code %v, got %v", tc.expectedCode, st.Code())
				}
			}
		})
	}
}

// --- AUTH TESTS ---

// startTestGRPCServer spins up a gRPC server with AuthInterceptor for testing.

func TestUpdateNode(t *testing.T) {
	tests := []struct {
		name            string
		updateNodeFunc  func(ctx context.Context, workflowID string, node *pb.Node) error
		expectedSuccess bool
		expectedCode    codes.Code
	}{
		{
			name: "success",
			updateNodeFunc: func(ctx context.Context, workflowID string, node *pb.Node) error {
				return nil
			},
			expectedSuccess: true,
			expectedCode:    codes.OK,
		},
		{
			name: "workflow not found",
			updateNodeFunc: func(ctx context.Context, workflowID string, node *pb.Node) error {
				return persistence.ErrWorkflowNotFound
			},
			expectedSuccess: false,
			expectedCode:    codes.NotFound,
		},
		{
			name: "internal error",
			updateNodeFunc: func(ctx context.Context, workflowID string, node *pb.Node) error {
				return errors.New("db failure")
			},
			expectedSuccess: false,
			expectedCode:    codes.Internal,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			svc := &WorkflowServiceServerImpl{
				StateManager: &fakeStateManager{
					UpdateNodeFunc: tc.updateNodeFunc,
				},
			}

			req := &pb.UpdateNodeRequest{
				WorkflowId: "wf-123",
				Node: &pb.Node{
					NodeId: "node-1",
				},
			}

			resp, err := svc.UpdateNode(context.Background(), req)
			if tc.expectedCode == codes.OK {
				if err != nil {
					t.Fatalf("expected no error, got %v", err)
				}
				if resp == nil || resp.Success != tc.expectedSuccess {
					t.Fatalf("expected success=%v, got %v", tc.expectedSuccess, resp)
				}
			} else {
				if err == nil {
					t.Fatalf("expected error code %v, got nil", tc.expectedCode)
				}
				st, ok := status.FromError(err)
				if !ok {
					t.Fatalf("expected gRPC status error, got %v", err)
				}
				if st.Code() != tc.expectedCode {
					t.Fatalf("expected code %v, got %v", tc.expectedCode, st.Code())
				}
				if resp == nil || resp.Success != tc.expectedSuccess {
					t.Fatalf("expected success=%v, got %v", tc.expectedSuccess, resp)
				}
			}
		})
	}
}

type FakeEventLogger struct {
	Events []Event
}

func (m *FakeEventLogger) LogEvent(e Event) {
	m.Events = append(m.Events, e)
}

func TestCreateWorkflow_EmitsEvent(t *testing.T) {
	fakeSM := &fakeStateManager{
		CreateWorkflowFunc: func(ctx context.Context, workflow *persistence.Workflow) (string, error) {
			return "generated-id", nil
		},
	}
	fakeLogger := &FakeEventLogger{}

	server := NewWorkflowServiceServer(fakeSM, fakeLogger)

	_, err := server.CreateWorkflow(context.Background(), &pb.CreateWorkflowRequest{})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(fakeLogger.Events) == 0 {
		t.Errorf("expected at least one event emitted, got none")
	} else if fakeLogger.Events[0].Type != EventWorkflowCreated {
		t.Errorf("expected event type %s, got %s", EventWorkflowCreated, fakeLogger.Events[0].Type)
	}
}

func TestUpdateNode_EmitsCorrectEvents(t *testing.T) {
	tests := []struct {
		name       string
		status     pb.Status
		expectType EventType
	}{
		{"dispatched", pb.Status_RUNNING, EventNodeDispatched},
		{"completed_pass", pb.Status_PASS, EventNodeCompleted},
		{"completed_fail", pb.Status_FAIL, EventNodeCompleted},
		{"completed_skipped", pb.Status_SKIPPED, EventNodeCompleted},
		{"completed_filtered", pb.Status_FILTERED, EventNodeCompleted},
		{"completed_task_error", pb.Status_TASK_ERROR, EventNodeCompleted},
		{"completed_infra_error", pb.Status_INFRA_ERROR, EventNodeCompleted},
		{"completed_timeout", pb.Status_TIMEOUT, EventNodeCompleted},
		{"completed_crash", pb.Status_CRASH, EventNodeCompleted},
		{"updated", pb.Status_UNKNOWN, EventNodeUpdated},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			fakeLogger := &FakeEventLogger{}
			svc := &WorkflowServiceServerImpl{
				StateManager: &fakeStateManager{
					UpdateNodeFunc: func(ctx context.Context, workflowID string, node *pb.Node) error {
						return nil
					},
				},
				EventLogger: fakeLogger,
			}
			req := &pb.UpdateNodeRequest{
				WorkflowId: "wf-1",
				Node: &pb.Node{
					NodeId: "node-1",
					Status: tc.status,
				},
			}
			_, err := svc.UpdateNode(context.Background(), req)
			if err != nil {
				t.Fatalf("expected no error, got %v", err)
			}
			if len(fakeLogger.Events) == 0 {
				t.Fatalf("expected event to be emitted")
			}
			gotType := fakeLogger.Events[0].Type
			if gotType != tc.expectType {
				t.Errorf("expected event type %s, got %s", tc.expectType, gotType)
			}
		})
	}
}
