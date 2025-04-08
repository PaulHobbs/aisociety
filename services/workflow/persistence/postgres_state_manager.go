package persistence

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/pgxpool"
	"google.golang.org/protobuf/proto"

	pb "paul.hobbs.page/aisociety/protos"
)

// PostgresStateManager manages workflow state in Postgres
type PostgresStateManager struct {
	pool *pgxpool.Pool
}

// NewPostgresStateManager creates a new PostgresStateManager
func NewPostgresStateManager(pool *pgxpool.Pool) *PostgresStateManager {
	return &PostgresStateManager{pool: pool}
}

// NewPostgresStateManagerFromConnStr creates a new PostgresStateManager from context and connection string

// ApplyNodeEdits applies a set of node edits to the workflow
func (p *PostgresStateManager) ApplyNodeEdits(ctx context.Context, workflowID string, edits []*pb.NodeEdit) error {
	tx, err := p.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}

	defer tx.Rollback(ctx)

	for _, edit := range edits {
		var applyErr error
		switch edit.Type {
		case pb.NodeEdit_INSERT:
			applyErr = p.applyInsertEdit(ctx, tx, workflowID, edit)
		case pb.NodeEdit_UPDATE:
			applyErr = p.applyUpdateEdit(ctx, tx, workflowID, edit)
		case pb.NodeEdit_DELETE:
			applyErr = p.applyDeleteEdit(ctx, tx, workflowID, edit)
		default:
			return fmt.Errorf("unknown edit type: %v", edit.Type)
		}
		if applyErr != nil {
			return applyErr
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

func (p *PostgresStateManager) applyInsertEdit(ctx context.Context, tx pgx.Tx, workflowID string, edit *pb.NodeEdit) error {
	if edit.Node == nil {
		return fmt.Errorf("node is required for INSERT edit")
	}
	if err := p.createNodeTx(ctx, tx, workflowID, edit.Node); err != nil {
		return fmt.Errorf("failed to apply INSERT edit: %w", err)
	}
	return nil
}

func (p *PostgresStateManager) applyUpdateEdit(ctx context.Context, tx pgx.Tx, workflowID string, edit *pb.NodeEdit) error {
	if edit.Node == nil {
		return fmt.Errorf("node is required for UPDATE edit")
	}

	nodeBytes, allTasksBytes, editsBytes, err := serializeNodeData(edit)
	if err != nil {
		return err
	}

	if err := updateNodeRecord(ctx, tx, workflowID, edit, nodeBytes, allTasksBytes, editsBytes); err != nil {
		return err
	}

	if err := replaceNodeEdges(ctx, tx, workflowID, edit); err != nil {
		return err
	}

	return nil
}

func (p *PostgresStateManager) applyDeleteEdit(ctx context.Context, tx pgx.Tx, workflowID string, edit *pb.NodeEdit) error {
	if edit.Node == nil || edit.Node.NodeId == "" {
		return fmt.Errorf("node ID is required for DELETE edit")
	}

	if err := deleteNodeEdges(ctx, tx, workflowID, edit.Node.NodeId); err != nil {
		return err
	}

	result, err := tx.Exec(ctx,
		`DELETE FROM nodes WHERE workflow_id = $1 AND id = $2`,
		workflowID, edit.Node.NodeId)
	if err != nil {
		return fmt.Errorf("failed to apply DELETE edit: %w", err)
	}
	if result.RowsAffected() == 0 {
		return fmt.Errorf("node not found for DELETE: %s", edit.Node.NodeId)
	}

	return nil
}

func serializeNodeData(edit *pb.NodeEdit) ([]byte, []byte, []byte, error) {
	nodeBytes, err := proto.Marshal(edit.Node)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to serialize node for UPDATE: %w", err)
	}

	var allTasksBytes []byte
	if len(edit.Node.AllTasks) > 0 {
		tasksMsg := &pb.TaskList{Tasks: edit.Node.AllTasks}
		allTasksBytes, err = proto.Marshal(tasksMsg)
		if err != nil {
			return nil, nil, nil, fmt.Errorf("failed to serialize all_tasks for UPDATE: %w", err)
		}
	}

	var editsBytes []byte
	if len(edit.Node.Edits) > 0 {
		editsMsg := &pb.NodeEditList{Edits: edit.Node.Edits}
		editsBytes, err = proto.Marshal(editsMsg)
		if err != nil {
			return nil, nil, nil, fmt.Errorf("failed to serialize edits for UPDATE: %w", err)
		}
	}

	return nodeBytes, allTasksBytes, editsBytes, nil
}

func updateNodeRecord(ctx context.Context, tx pgx.Tx, workflowID string, edit *pb.NodeEdit, nodeBytes, allTasksBytes, editsBytes []byte) error {
	result, err := tx.Exec(ctx,
		`UPDATE nodes SET status = $1, node = $2, all_tasks = $3, edits = $4, updated_at = $5
         WHERE workflow_id = $6 AND id = $7`,
		int(edit.Node.Status), nodeBytes, allTasksBytes, editsBytes, time.Now(), workflowID, edit.Node.NodeId)
	if err != nil {
		return fmt.Errorf("failed to apply UPDATE edit: %w", err)
	}
	if result.RowsAffected() == 0 {
		return fmt.Errorf("node not found for UPDATE: %s", edit.Node.NodeId)
	}
	return nil
}

func replaceNodeEdges(ctx context.Context, tx pgx.Tx, workflowID string, edit *pb.NodeEdit) error {
	if err := deleteNodeEdges(ctx, tx, workflowID, edit.Node.NodeId); err != nil {
		return fmt.Errorf("failed to delete existing edges for UPDATE: %w", err)
	}

	// Batch insert parent edges
	if len(edit.Node.ParentIds) > 0 {
		if err := batchInsertEdges(ctx, tx, workflowID, edit.Node.ParentIds, edit.Node.NodeId, true); err != nil {
			return fmt.Errorf("failed to insert parent edges for UPDATE: %w", err)
		}
	}

	// Batch insert child edges
	if len(edit.Node.ChildIds) > 0 {
		if err := batchInsertEdges(ctx, tx, workflowID, []string{edit.Node.NodeId}, edit.Node.ChildIds, false); err != nil {
			return fmt.Errorf("failed to insert child edges for UPDATE: %w", err)
		}
	}

	return nil
}

func deleteNodeEdges(ctx context.Context, tx pgx.Tx, workflowID, nodeID string) error {
	_, err := tx.Exec(ctx,
		`DELETE FROM node_edges WHERE workflow_id = $1 AND (parent_node_id = $2 OR child_node_id = $2)`,
		workflowID, nodeID)
	if err != nil {
		return fmt.Errorf("failed to delete node edges: %w", err)
	}
	return nil
}
func (p *PostgresStateManager) createNodeTx(ctx context.Context, tx pgx.Tx, workflowID string, node *pb.Node) error {
	nodeBytes, allTasksBytes, editsBytes, err := serializeNodeData(&pb.NodeEdit{Node: node})
	if err != nil {
		return err
	}

	_, err = tx.Exec(ctx,
		`INSERT INTO nodes (workflow_id, id, status, node, all_tasks, edits, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $7)`,
		workflowID, node.NodeId, int(node.Status), nodeBytes, allTasksBytes, editsBytes, time.Now(),
	)
	if err != nil {
		return fmt.Errorf("failed to insert node: %w", err)
	}

	// Insert parent edges
	if len(node.ParentIds) > 0 {
		if err := batchInsertEdges(ctx, tx, workflowID, node.ParentIds, node.NodeId, true); err != nil {
			return fmt.Errorf("failed to insert parent edges: %w", err)
		}
	}

	// Insert child edges
	if len(node.ChildIds) > 0 {
		if err := batchInsertEdges(ctx, tx, workflowID, []string{node.NodeId}, node.ChildIds, false); err != nil {
			return fmt.Errorf("failed to insert child edges: %w", err)
		}
	}

	return nil
}

// batchInsertEdges inserts multiple edges efficiently
// If isParentInsert is true, parentIDs are used as parent_node_id, nodeID as child_node_id
// If false, nodeID is parent_node_id, childIDs as child_node_id
func batchInsertEdges(ctx context.Context, tx pgx.Tx, workflowID string, ids1 interface{}, ids2 interface{}, isParentInsert bool) error {
	var values []interface{}
	var idx = 1
	var placeholdersBuilder strings.Builder

	switch {
	case isParentInsert:
		parentIDs := ids1.([]string)
		var placeholdersBuilder strings.Builder

		appendPlaceholder := func(idx int) {
			fmt.Fprintf(&placeholdersBuilder, "($%d, $%d, $%d)", idx, idx+1, idx+2)
		}

		switch {
		case isParentInsert:
			childID, ok := ids2.(string)
			if !ok {
				return fmt.Errorf("expected ids2 to be string, got %T", ids2)
			}
			for _, parentID := range parentIDs {
				if placeholdersBuilder.Len() > 0 {
					placeholdersBuilder.WriteString(",")
				}
				appendPlaceholder(idx)
				values = append(values, workflowID, parentID, childID)
				idx += 3
			}

		case !isParentInsert:
			parentIDsSlice, ok := ids1.([]string)
			if !ok || len(parentIDsSlice) == 0 {
				return fmt.Errorf("expected ids1 to be non-empty []string, got %T", ids1)
			}
			parentID := parentIDsSlice[0]

			childIDs, ok := ids2.([]string)
			if !ok {
				return fmt.Errorf("expected ids2 to be []string, got %T", ids2)
			}
			for _, childID := range childIDs {
				if placeholdersBuilder.Len() > 0 {
					placeholdersBuilder.WriteString(",")
				}
				appendPlaceholder(idx)
				values = append(values, workflowID, parentID, childID)
				idx += 3
			}
		}
	}

	if placeholdersBuilder.Len() == 0 {
		return nil
	}

	query := `INSERT INTO node_edges (workflow_id, parent_node_id, child_node_id) VALUES ` + placeholdersBuilder.String()

	_, err := tx.Exec(ctx, query, values...)
	if err != nil {
		return err
	}
	return nil
}

// NewPostgresStateManagerFromConnStr creates a new PostgresStateManager from context and connection string
func NewPostgresStateManagerFromConnStr(ctx context.Context, connStr string) (*PostgresStateManager, error) {
	pool, err := pgxpool.Connect(ctx, connStr)
	if err != nil {
		return nil, fmt.Errorf("failed to create pgx pool: %w", err)
	}
	return &PostgresStateManager{pool: pool}, nil
}

// Close closes the underlying pgx pool
func (p *PostgresStateManager) Close() {
	if p.pool != nil {
		p.pool.Close()
	}
}
func (p *PostgresStateManager) CreateWorkflow(ctx context.Context, workflow interface{}) error {
	wf, ok := workflow.(*Workflow)
	if !ok {
		return fmt.Errorf("expected *Workflow, got %T", workflow)
	}

	query := `INSERT INTO workflows (name, description, status) VALUES ($1, $2, $3) RETURNING id`
	err := p.pool.QueryRow(ctx, query, wf.Name, wf.Description, int32(wf.Status)).Scan(&wf.ID)
	if err != nil {
		return fmt.Errorf("CreateWorkflow insert failed: %w", err)
	}
	return nil
}

func (p *PostgresStateManager) GetWorkflow(ctx context.Context, workflowID string) (*Workflow, error) {
	query := `SELECT id, name, description, status FROM workflows WHERE id = $1`
	var wf Workflow
	var statusCode int32
	err := p.pool.QueryRow(ctx, query, workflowID).Scan(&wf.ID, &wf.Name, &wf.Description, &statusCode)
	if err != nil {
		// Return (nil, nil) if no workflow found
		if err.Error() == "no rows in result set" {
			return nil, nil
		}
		return nil, fmt.Errorf("GetWorkflow query failed: %w", err)
	}
	wf.Status = pb.Status(statusCode)
	return &wf, nil
}

func (p *PostgresStateManager) CreateNode(ctx context.Context, workflowID string, node *pb.Node) error {
	nodeBytes, err := proto.Marshal(node)
	if err != nil {
		return fmt.Errorf("failed to marshal node proto: %w", err)
	}

	query := `INSERT INTO nodes (workflow_id, node_id, status, node) VALUES ($1, $2, $3, $4)`
	_, err = p.pool.Exec(ctx, query, workflowID, node.NodeId, int32(node.Status), nodeBytes)
	if err != nil {
		return fmt.Errorf("CreateNode insert failed: %w", err)
	}
	return nil
}

func (p *PostgresStateManager) GetNode(ctx context.Context, workflowID, nodeID string) (*pb.Node, error) {
	query := `SELECT node FROM nodes WHERE workflow_id = $1 AND node_id = $2`
	var nodeBytes []byte
	err := p.pool.QueryRow(ctx, query, workflowID, nodeID).Scan(&nodeBytes)
	if err != nil {
		if err.Error() == "no rows in result set" {
			return nil, nil
		}
		return nil, fmt.Errorf("GetNode query failed: %w", err)
	}

	var node pb.Node
	err = proto.Unmarshal(nodeBytes, &node)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal node proto: %w", err)
	}
	return &node, nil
}
