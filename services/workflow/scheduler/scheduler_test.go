package scheduler

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	pb "paul.hobbs.page/aisociety/protos"
)

// FakeStateManager implements StateManager for testing.
type FakeStateManager struct {
	mu           sync.Mutex
	readyNodes   []*pb.Node
	updatedNodes []*pb.Node
	appliedEdits [][]*pb.NodeEdit
}

func (m *FakeStateManager) FindReadyNodes(ctx context.Context) ([]*pb.Node, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.readyNodes, nil
}

func (m *FakeStateManager) UpdateNode(ctx context.Context, workflowID string, node *pb.Node) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.updatedNodes = append(m.updatedNodes, node)
	return nil
}

func (m *FakeStateManager) ApplyNodeEdits(ctx context.Context, workflowID string, edits []*pb.NodeEdit) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.appliedEdits = append(m.appliedEdits, edits)
	return nil
}

// FakeNodeServiceClient implements NodeServiceClient for testing.
type FakeNodeServiceClient struct {
	Response *pb.ExecuteNodeResponse
	Err      error
	Called   bool
}

func (m *FakeNodeServiceClient) ExecuteNode(ctx context.Context, req *pb.ExecuteNodeRequest) (*pb.ExecuteNodeResponse, error) {
	m.Called = true
	return m.Response, m.Err
}

func TestSchedulerDispatchesReadyNodes(t *testing.T) {
	fakeSM := &FakeStateManager{
		readyNodes: []*pb.Node{
			{NodeId: "node1", Status: pb.Status_BLOCKED}, // Use BLOCKED (9) as the initial state
		},
	}
	fakeClient := &FakeNodeServiceClient{
		Response: &pb.ExecuteNodeResponse{
			Node: &pb.Node{
				NodeId: "node1",
				Status: pb.Status_PASS,
			},
		},
	}
	sched := NewSimpleScheduler(fakeSM, fakeClient, 10*time.Millisecond)

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	go sched.Run(ctx)

	// Wait up to 50ms for fakeClient.Called to become true
	waitCtx, waitCancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer waitCancel()

	for {
		if fakeClient.Called {
			break
		}
		select {
		case <-waitCtx.Done():
			break
		default:
			time.Sleep(1 * time.Millisecond)
		}
	}

	if !fakeClient.Called {
		t.Errorf("Expected ExecuteNode to be called")
	}

	fakeSM.mu.Lock()
	defer fakeSM.mu.Unlock()
	if len(fakeSM.updatedNodes) == 0 {
		t.Errorf("Expected node status to be updated")
	}
}

func TestSchedulerHandlesNodeServiceError(t *testing.T) {
	foundInfraError := false

	fakeSM := &FakeStateManager{
		readyNodes: []*pb.Node{
			{NodeId: "node2", Status: pb.Status_BLOCKED}, // Use BLOCKED (9) as the initial state
		},
	}
	fakeClient := &FakeNodeServiceClient{
		Err: errors.New("node service error"),
	}
	sched := NewSimpleScheduler(fakeSM, fakeClient, 10*time.Millisecond)

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	go sched.Run(ctx)

	timeout := time.After(200 * time.Millisecond)
	tick := time.Tick(1 * time.Millisecond)
pollLoop:
	for {
		select {
		case <-timeout:
			break pollLoop
		case <-tick:
			fakeSM.mu.Lock()
			for _, n := range fakeSM.updatedNodes {
				if n.Status == pb.Status_INFRA_ERROR {
					foundInfraError = true
					fakeSM.mu.Unlock()
					break pollLoop
				}
			}
			fakeSM.mu.Unlock()
		}
	}

	if !fakeClient.Called {
		t.Errorf("Expected ExecuteNode to be called")
	}

	fakeSM.mu.Lock()
	defer fakeSM.mu.Unlock()
	foundInfraError = false
	for _, n := range fakeSM.updatedNodes {
		if n.Status == pb.Status_INFRA_ERROR {
			foundInfraError = true
			break
		}
	}
	if !foundInfraError {
		t.Errorf("Expected node status to be INFRA_ERROR on dispatch failure")
	}
}

func TestSchedulerAppliesNodeEdits(t *testing.T) {
	edit := &pb.NodeEdit{Description: "test edit"}

	fakeSM := &FakeStateManager{
		readyNodes: []*pb.Node{
			{NodeId: "node3", Status: pb.Status_BLOCKED},
		},
	}

	fakeClient := &FakeNodeServiceClient{
		Response: &pb.ExecuteNodeResponse{
			Node: &pb.Node{
				NodeId: "node3",
				Status: pb.Status_PASS,
				Edits:  []*pb.NodeEdit{edit},
			},
		},
	}

	scheduler := NewSimpleScheduler(fakeSM, fakeClient, 10*time.Millisecond)

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	done := make(chan struct{})
	go func() {
		scheduler.Run(ctx)
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(60 * time.Millisecond):
		t.Fatal("Scheduler did not finish in time")
	}

	fakeSM.mu.Lock()
	defer fakeSM.mu.Unlock()
	if len(fakeSM.appliedEdits) == 0 {
		t.Errorf("Expected NodeEdits to be applied")
	}
}

func FuzzSchedulerAppliesNodeEdits(f *testing.F) {
	f.Add("initial description", int32(pb.Status_BLOCKED), "edit description", int32(pb.Status_PASS))

	f.Fuzz(func(t *testing.T, initialDesc string, initialStatusInt int32, editDesc string, finalStatusInt int32) {
		initialStatus := pb.Status(initialStatusInt)
		finalStatus := pb.Status(finalStatusInt)

		edit := &pb.NodeEdit{Description: editDesc}

		fakeSM := &FakeStateManager{
			readyNodes: []*pb.Node{
				{NodeId: "nodeFuzz", Status: initialStatus, Description: initialDesc},
			},
		}

		fakeClient := &FakeNodeServiceClient{
			Response: &pb.ExecuteNodeResponse{
				Node: &pb.Node{
					NodeId: "nodeFuzz",
					Status: finalStatus,
					Edits:  []*pb.NodeEdit{edit},
				},
			},
		}

		scheduler := NewSimpleScheduler(fakeSM, fakeClient, 1*time.Millisecond)

		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
		defer cancel()

		done := make(chan struct{})
		go func() {
			scheduler.Run(ctx)
			close(done)
		}()

		select {
		case <-done:
		case <-time.After(1000 * time.Millisecond):
			t.Fatal("Scheduler did not finish in time")
		}

		fakeSM.mu.Lock()
		defer fakeSM.mu.Unlock()
		if len(fakeSM.appliedEdits) == 0 {
			t.Errorf("Expected NodeEdits to be applied")
		}
	})
}
