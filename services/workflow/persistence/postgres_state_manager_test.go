package persistence

import (
	"context"
	"math/rand"
	"os"
	"testing"

	pb "paul.hobbs.page/aisociety/protos"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v4"
	"google.golang.org/protobuf/proto"
)

var testManager *PostgresStateManager

func TestMain(m *testing.M) {
	connStr := os.Getenv("TEST_DATABASE_URL")
	if connStr == "" {
		connStr = "postgres://aisociety:aTEST_DATABASE_isociety@localhost:55433/aisociety_test_db?sslmode=disable"
	}

	ctx := context.Background()
	var err error
	testManager, err = NewPostgresStateManagerFromConnStr(ctx, connStr)
	if err != nil {
		panic(err)
	}
	defer testManager.Close()

	code := m.Run()
	os.Exit(code)
}

func cleanDB(t *testing.T) {
	connStr := os.Getenv("TEST_DATABASE_URL")
	conn, err := pgx.Connect(context.Background(), connStr)
	if err != nil {
		t.Fatalf("failed to connect for cleanup: %v", err)
	}
	defer conn.Close(context.Background())

	_, err = conn.Exec(context.Background(), "TRUNCATE node_edges, nodes, workflows RESTART IDENTITY CASCADE;")
	if err != nil {
		t.Fatalf("failed to clean db: %v", err)
	}
}

func TestCreateAndGetWorkflow(t *testing.T) {
	cleanDB(t)

	ctx := context.Background()
	workflow := &Workflow{
		Name:        "Test Workflow",
		Description: "A workflow for testing",
		Status:      pb.Status_UNKNOWN,
	}

	_, err := testManager.CreateWorkflow(ctx, workflow)
	if err != nil {
		t.Fatalf("CreateWorkflow failed: %v", err)
	}

	got, err := testManager.GetWorkflow(ctx, workflow.ID)
	if err != nil {
		t.Fatalf("GetWorkflow failed: %v", err)
	}

	if got.Name != workflow.Name || got.Description != workflow.Description || got.Status != workflow.Status {
		t.Errorf("Got workflow %+v, want %+v", got, workflow)
	}
}

func TestCreateAndGetNode(t *testing.T) {
	cleanDB(t)

	ctx := context.Background()
	workflow := &Workflow{
		Name:        "Node Test Workflow",
		Description: "Workflow for node test",
		Status:      pb.Status_UNKNOWN,
	}

	_, err := testManager.CreateWorkflow(ctx, workflow)
	if err != nil {
		t.Fatalf("CreateWorkflow failed: %v", err)
	}

	node := &pb.Node{
		NodeId:      uuid.New().String(),
		Description: "Test Node",
	}

	err = testManager.CreateNode(ctx, workflow.ID, node)
	if err != nil {
		t.Fatalf("CreateNode failed: %v", err)
	}

	gotNode, err := testManager.GetNode(ctx, workflow.ID, node.NodeId)
	if err != nil {
		t.Fatalf("GetNode failed: %v", err)
	}

	if !proto.Equal(gotNode, node) {
		t.Errorf("Got node %+v, want %+v", gotNode, node)
	}
	// --- Test UpdateNode ---
	node.Description = "Updated Description"
	node.Status = pb.Status_RUNNING

	err = testManager.UpdateNode(ctx, workflow.ID, node)
	if err != nil {
		t.Fatalf("UpdateNode failed: %v", err)
	}

	updatedNode, err := testManager.GetNode(ctx, workflow.ID, node.NodeId)
	if err != nil {
		t.Fatalf("GetNode after update failed: %v", err)
	}

	if !proto.Equal(updatedNode, node) {
		t.Errorf("After update, got node %+v, want %+v", updatedNode, node)
	}
}
func TestListWorkflows(t *testing.T) {
	cleanDB(t)
	ctx := context.Background()

	// Initially, should be empty
	ids, err := testManager.ListWorkflows(ctx)
	if err != nil {
		t.Fatalf("ListWorkflows failed: %v", err)
	}
	if len(ids) != 0 {
		t.Errorf("Expected no workflows, got %d", len(ids))
	}

	// Add two workflows
	wf1 := &Workflow{Name: "WF1", Description: "desc1", Status: pb.Status_UNKNOWN}
	wf2 := &Workflow{Name: "WF2", Description: "desc2", Status: pb.Status_UNKNOWN}
	id1, err := testManager.CreateWorkflow(ctx, wf1)
	if err != nil {
		t.Fatalf("CreateWorkflow 1 failed: %v", err)
	}
	id2, err := testManager.CreateWorkflow(ctx, wf2)
	if err != nil {
		t.Fatalf("CreateWorkflow 2 failed: %v", err)
	}

	ids, err = testManager.ListWorkflows(ctx)
	if err != nil {
		t.Fatalf("ListWorkflows after insert failed: %v", err)
	}
	if len(ids) != 2 {
		t.Errorf("Expected 2 workflows, got %d", len(ids))
	}
	found1, found2 := false, false
	for _, id := range ids {
		if id == id1 {
			found1 = true
		}
		if id == id2 {
			found2 = true
		}
	}
	if !found1 || !found2 {
		t.Errorf("Workflow IDs not found in list: got %v, want %v and %v", ids, id1, id2)
	}
}

func TestGetWorkflow_NotFound(t *testing.T) {
	cleanDB(t)
	ctx := context.Background()
	_, err := testManager.GetWorkflow(ctx, "nonexistent-id")
	if err == nil {
		t.Errorf("Expected error for nonexistent workflow, got nil")
	}
}

func TestGetNode_NotFound(t *testing.T) {
	cleanDB(t)
	ctx := context.Background()
	_, err := testManager.GetNode(ctx, "nonexistent-wf", "nonexistent-node")
	if err == nil {
		t.Errorf("Expected error for nonexistent node, got nil")
	}
}

func TestUpdateNode_NotFound(t *testing.T) {
	cleanDB(t)
	ctx := context.Background()
	node := &pb.Node{
		NodeId:      uuid.New().String(),
		Description: "Should not exist",
	}
	err := testManager.UpdateNode(ctx, "nonexistent-wf", node)
	if err == nil {
		t.Errorf("Expected error for updating nonexistent node, got nil")
	}
}

func TestCreateNode_InvalidWorkflow(t *testing.T) {
	cleanDB(t)
	ctx := context.Background()
	node := &pb.Node{
		NodeId:      uuid.New().String(),
		Description: "Orphan node",
	}
	err := testManager.CreateNode(ctx, "nonexistent-wf", node)
	if err == nil {
		t.Errorf("Expected error for creating node in nonexistent workflow, got nil")
	}
}

func TestFindReadyNodes(t *testing.T) {
	cleanDB(t)
	ctx := context.Background()
	// No workflows/nodes: should return empty
	nodes, err := testManager.FindReadyNodes(ctx)
	if err != nil {
		t.Fatalf("FindReadyNodes failed: %v", err)
	}
	if len(nodes) != 0 {
		t.Errorf("Expected no ready nodes, got %d", len(nodes))
	}

	// Create a workflow and a node with Status_READY
	wf := &Workflow{Name: "ReadyNodesWF", Description: "desc", Status: pb.Status_UNKNOWN}
	_, err = testManager.CreateWorkflow(ctx, wf)
	if err != nil {
		t.Fatalf("CreateWorkflow failed: %v", err)
	}
	node := &pb.Node{
		NodeId:      uuid.New().String(),
		Description: "Ready Node",
		Status:      pb.Status_RUNNING, // Not ready
	}
	err = testManager.CreateNode(ctx, wf.ID, node)
	if err != nil {
		t.Fatalf("CreateNode failed: %v", err)
	}
	// Update node to READY
	node.Status = pb.Status_PASS // Let's assume PASS is considered ready for this test
	err = testManager.UpdateNode(ctx, wf.ID, node)
	if err != nil {
		t.Fatalf("UpdateNode failed: %v", err)
	}
	// Now FindReadyNodes should return at least one node
	nodes, err = testManager.FindReadyNodes(ctx)
	if err != nil {
		t.Fatalf("FindReadyNodes after insert failed: %v", err)
	}
	if len(nodes) == 0 {
		t.Errorf("Expected at least one ready node, got 0")
	}
}

func TestApplyNodeEdits_Unit(t *testing.T) {
	cleanDB(t)
	ctx := context.Background()
	wf := &Workflow{Name: "ApplyEditsWF", Description: "desc", Status: pb.Status_UNKNOWN}
	_, err := testManager.CreateWorkflow(ctx, wf)
	if err != nil {
		t.Fatalf("CreateWorkflow failed: %v", err)
	}
	node := &pb.Node{
		NodeId:      uuid.New().String(),
		Description: "Node for edits",
	}
	// Insert node
	editInsert := &pb.NodeEdit{
		Type: pb.NodeEdit_INSERT,
		Node: node,
	}
	err = testManager.ApplyNodeEdits(ctx, wf.ID, []*pb.NodeEdit{editInsert})
	if err != nil {
		t.Fatalf("ApplyNodeEdits INSERT failed: %v", err)
	}
	// Update node
	node.Description = "Edited"
	editUpdate := &pb.NodeEdit{
		Type: pb.NodeEdit_UPDATE,
		Node: node,
	}
	err = testManager.ApplyNodeEdits(ctx, wf.ID, []*pb.NodeEdit{editUpdate})
	if err != nil {
		t.Fatalf("ApplyNodeEdits UPDATE failed: %v", err)
	}
	got, err := testManager.GetNode(ctx, wf.ID, node.NodeId)
	if err != nil {
		t.Fatalf("GetNode after update failed: %v", err)
	}
	if got.Description != "Edited" {
		t.Errorf("Node description not updated, got %q", got.Description)
	}
	// Delete node
	editDelete := &pb.NodeEdit{
		Type: pb.NodeEdit_DELETE,
		Node: &pb.Node{NodeId: node.NodeId},
	}
	err = testManager.ApplyNodeEdits(ctx, wf.ID, []*pb.NodeEdit{editDelete})
	if err != nil {
		t.Fatalf("ApplyNodeEdits DELETE failed: %v", err)
	}
	_, err = testManager.GetNode(ctx, wf.ID, node.NodeId)
	if err == nil {
		t.Errorf("Expected error after deleting node, got nil")
	}
	// Edge case: apply edit to nonexistent workflow
	err = testManager.ApplyNodeEdits(ctx, "nonexistent-wf", []*pb.NodeEdit{editInsert})
	if err == nil {
		t.Errorf("Expected error for ApplyNodeEdits on nonexistent workflow, got nil")
	}
}

func TestClose_Idempotent(t *testing.T) {
	// Just ensure Close can be called multiple times without panic
	err := testManager.Close()
	if err != nil {
		t.Errorf("Close failed: %v", err)
	}
	// Should be safe to call again
	err = testManager.Close()
	if err != nil {
		t.Errorf("Second Close failed: %v", err)
	}
}

func FuzzApplyNodeEdits(f *testing.F) {
	// Clean database once before fuzzing starts
	cleanDB(&testing.T{})
	f.Fuzz(func(t *testing.T, seed int64) {
		r := rand.New(rand.NewSource(seed))
		ctx := context.Background()

		workflow := &Workflow{
			Name:        "Fuzz Workflow",
			Description: "Workflow for fuzzing",
			Status:      pb.Status_UNKNOWN,
		}
		_, err := testManager.CreateWorkflow(ctx, workflow)
		if err != nil {
			t.Fatalf("CreateWorkflow failed: %v", err)
		}

		// Track existing node IDs to create realistic parent/child references
		var existingIDs []string

		// Generate 1-10 random edits
		numEdits := r.Intn(10) + 1
		var edits []*pb.NodeEdit
		for i := 0; i < numEdits; i++ {
			edit := randomNodeEdit(r, existingIDs)
			if edit.Node != nil && edit.Node.NodeId != "" {
				existingIDs = append(existingIDs, edit.Node.NodeId)
			}
			edits = append(edits, edit)
		}

		err = testManager.ApplyNodeEdits(ctx, workflow.ID, edits)
		if err != nil && !isExpectedError(err) {
			t.Errorf("ApplyNodeEdits error: %v", err)
		}
	})
}

func randomNodeEdit(r *rand.Rand, existingIDs []string) *pb.NodeEdit {
	editTypes := []pb.NodeEdit_Type{
		pb.NodeEdit_INSERT,
		pb.NodeEdit_UPDATE,
		pb.NodeEdit_DELETE,
	}
	editType := editTypes[r.Intn(len(editTypes))]

	node := &pb.Node{}
	if editType == pb.NodeEdit_DELETE {
		// For delete, pick existing or random ID
		if len(existingIDs) > 0 && r.Float32() < 0.7 {
			node.NodeId = existingIDs[r.Intn(len(existingIDs))]
		} else {
			node.NodeId = uuid.New().String()
		}
	} else {
		node = randomNode(r, existingIDs, 0)
	}

	return &pb.NodeEdit{
		Type:        editType,
		Timestamp:   nil, // can add timestamp if needed
		Description: randomString(r, 20),
		Node:        node,
	}
}

func randomNode(r *rand.Rand, existingIDs []string, depth int) *pb.Node {
	id := uuid.New().String()

	// Randomly select some existing IDs as parents/children
	var parents, children []string
	if len(existingIDs) > 0 {
		for _, idList := range []*[]string{&parents, &children} {
			n := r.Intn(3)
			for i := 0; i < n; i++ {
				*idList = append(*idList, existingIDs[r.Intn(len(existingIDs))])
			}
		}
	}

	// Random nested tasks
	var allTasks []*pb.Task
	numTasks := r.Intn(3)
	for i := 0; i < numTasks; i++ {
		allTasks = append(allTasks, randomTask(r, 0))
	}

	// Random nested edits (limit recursion depth)
	var nestedEdits []*pb.NodeEdit
	if depth < 1 && r.Float32() < 0.3 {
		numNested := r.Intn(2)
		for i := 0; i < numNested; i++ {
			nestedEdits = append(nestedEdits, randomNodeEdit(r, existingIDs))
		}
	}

	return &pb.Node{
		NodeId:      id,
		Description: randomString(r, 30),
		ParentIds:   parents,
		ChildIds:    children,
		Agent:       &pb.Agent{AgentId: uuid.New().String(), Role: "Worker", ModelType: "GPT-4"},
		ExecutionOptions: &pb.ExecutionOptions{
			Timeout: nil,
			RetryOptions: &pb.ExecutionOptions_RetryOptions{
				MaxAttempts: int32(r.Intn(5)),
			},
		},
		AllTasks:     allTasks,
		AssignedTask: randomTask(r, 0),
		Status:       randomStatus(r),
		Edits:        nestedEdits,
		IsFinal:      r.Float32() < 0.5,
	}
}

func randomTask(r *rand.Rand, depth int) *pb.Task {
	task := &pb.Task{
		Id:            uuid.New().String(),
		Goal:          randomString(r, 20),
		DependencyIds: nil,
	}

	// Random dependencies
	numDeps := r.Intn(3)
	for i := 0; i < numDeps; i++ {
		task.DependencyIds = append(task.DependencyIds, uuid.New().String())
	}

	// Random results
	numResults := r.Intn(3)
	for i := 0; i < numResults; i++ {
		task.Results = append(task.Results, &pb.Task_Result{
			Status:    randomStatus(r),
			Summary:   randomString(r, 15),
			Output:    randomString(r, 50),
			Artifacts: map[string]string{"log": "http://example.com/log"},
		})
	}

	// Nested subtasks (limit depth)
	if depth < 1 && r.Float32() < 0.5 {
		numSub := r.Intn(2)
		for i := 0; i < numSub; i++ {
			task.Subtasks = append(task.Subtasks, randomTask(r, depth+1))
		}
	}

	return task
}

func randomStatus(r *rand.Rand) pb.Status {
	statuses := []pb.Status{
		pb.Status_UNKNOWN,
		pb.Status_PASS,
		pb.Status_FAIL,
		pb.Status_SKIPPED,
		pb.Status_FILTERED,
		pb.Status_TASK_ERROR,
		pb.Status_INFRA_ERROR,
		pb.Status_TIMEOUT,
		pb.Status_CRASH,
		pb.Status_BLOCKED,
		pb.Status_RUNNING,
	}
	return statuses[r.Intn(len(statuses))]
}

func randomString(r *rand.Rand, maxLen int) string {
	n := r.Intn(maxLen) + 1
	letters := []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789 ")
	b := make([]rune, n)
	for i := range b {
		b[i] = letters[r.Intn(len(letters))]
	}
	return string(b)
}

func isExpectedError(err error) bool {
	// Acceptable errors during fuzzing, e.g., constraint violations, missing nodes
	msg := err.Error()
	if msg == "" {
		return false
	}
	expectedSubstrings := []string{
		"node not found",
		"duplicate key",
		"violates foreign key constraint",
		"failed to apply DELETE edit",
		"failed to apply UPDATE edit",
		"failed to insert node",
	}
	for _, substr := range expectedSubstrings {
		if contains(msg, substr) {
			return true
		}
	}
	return false
}

func contains(s, substr string) bool {
	return len(substr) > 0 && len(s) >= len(substr) && (s == substr || len(s) > len(substr) && (s[:len(substr)] == substr || contains(s[1:], substr)))
}
