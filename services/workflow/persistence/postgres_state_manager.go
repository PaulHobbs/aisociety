package persistence

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/pgxpool"
	"google.golang.org/protobuf/proto"

	pb "paul.hobbs.page/aisociety/protos"
)

// PostgresStateManager implements the StateManager interface using PostgreSQL
type PostgresStateManager struct {
	pool *pgxpool.Pool
}

// NewPostgresStateManager creates a new PostgreSQL-backed state manager
func NewPostgresStateManager(ctx context.Context, connString string) (*PostgresStateManager, error) {
	pool, err := pgxpool.Connect(ctx, connString)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	// Verify connection
	if err := pool.Ping(ctx); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return &PostgresStateManager{
		pool: pool,
	}, nil
}

// Close closes the database connection pool
func (p *PostgresStateManager) Close() error {
	if p.pool != nil {
		p.pool.Close()
	}
	return nil
}

// CreateWorkflow persists a new workflow and its initial nodes to the database
func (p *PostgresStateManager) CreateWorkflow(ctx context.Context, workflow *Workflow) error {
	tx, err := p.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	// Generate a new UUID if not provided
	if workflow.ID == "" {
		workflow.ID = uuid.New().String()
	}

	// Insert workflow record
	_, err = tx.Exec(ctx,
		`INSERT INTO workflows (id, name, description, status) 
         VALUES ($1, $2, $3, $4)`,
		workflow.ID, workflow.Name, workflow.Description, int(workflow.Status))
	if err != nil {
		return fmt.Errorf("failed to insert workflow: %w", err)
	}

	// Insert nodes if provided
	for _, node := range workflow.Nodes {
		if err := p.createNodeTx(ctx, tx, workflow.ID, node); err != nil {
			return fmt.Errorf("failed to insert node: %w", err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// createNodeTx creates a node within an existing transaction
func (p *PostgresStateManager) createNodeTx(ctx context.Context, tx pgx.Tx, workflowID string, node *pb.Node) error {
	// Serialize the node protobuf
	nodeBytes, err := proto.Marshal(node)
	if err != nil {
		return fmt.Errorf("failed to serialize node: %w", err)
	}

	// Extract all_tasks and edits for separate storage
	var allTasksBytes, editsBytes []byte
	if len(node.AllTasks) > 0 {
		// Create a temporary message to hold just the tasks
		tasksMsg := &pb.TaskList{
			Tasks: node.AllTasks,
		}
		allTasksBytes, err = proto.Marshal(tasksMsg)
		if err != nil {
			return fmt.Errorf("failed to serialize all_tasks: %w", err)
		}
	}

	if len(node.Edits) > 0 {
		editsMsg := &pb.NodeEditList{
			Edits: node.Edits,
		}
		editsBytes, err = proto.Marshal(editsMsg)
		if err != nil {
			return fmt.Errorf("failed to serialize edits: %w", err)
		}
	}

	// Insert node record
	_, err = tx.Exec(ctx,
		`INSERT INTO nodes (id, workflow_id, node_id, status, node, all_tasks, edits)
		       VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		node.NodeId, workflowID, node.NodeId, int(node.Status), nodeBytes, allTasksBytes, editsBytes)
	if err != nil {
		return fmt.Errorf("failed to insert node: %w", err)
	}

	// Insert node edges for parent-child relationships
	for _, parentID := range node.ParentIds {
		_, err = tx.Exec(ctx,
			`INSERT INTO node_edges (workflow_id, parent_node_id, child_node_id) 
             VALUES ($1, $2, $3)`,
			workflowID, parentID, node.NodeId)
		if err != nil {
			return fmt.Errorf("failed to insert parent edge: %w", err)
		}
	}

	for _, childID := range node.ChildIds {
		_, err = tx.Exec(ctx,
			`INSERT INTO node_edges (workflow_id, parent_node_id, child_node_id) 
             VALUES ($1, $2, $3)`,
			workflowID, node.NodeId, childID)
		if err != nil {
			return fmt.Errorf("failed to insert child edge: %w", err)
		}
	}

	return nil
}

// GetWorkflow retrieves a workflow and all its nodes from the database
func (p *PostgresStateManager) GetWorkflow(ctx context.Context, workflowID string) (*Workflow, error) {
	// Query workflow record
	var workflow Workflow
	var statusInt int
	err := p.pool.QueryRow(ctx,
		`SELECT id, name, description, status FROM workflows WHERE id = $1`,
		workflowID).Scan(&workflow.ID, &workflow.Name, &workflow.Description, &statusInt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("workflow not found: %s", workflowID)
		}
		return nil, fmt.Errorf("failed to query workflow: %w", err)
	}
	workflow.Status = pb.Status(statusInt)

	// Query all nodes for this workflow
	rows, err := p.pool.Query(ctx,
		`SELECT id, status, node, all_tasks, edits FROM nodes WHERE workflow_id = $1`,
		workflowID)
	if err != nil {
		return nil, fmt.Errorf("failed to query nodes: %w", err)
	}
	defer rows.Close()

	// Process each node
	for rows.Next() {
		var nodeID string
		var statusInt int
		var nodeBytes, allTasksBytes, editsBytes []byte
		if err := rows.Scan(&nodeID, &statusInt, &nodeBytes, &allTasksBytes, &editsBytes); err != nil {
			return nil, fmt.Errorf("failed to scan node row: %w", err)
		}

		// Deserialize node protobuf
		node := &pb.Node{}
		if err := proto.Unmarshal(nodeBytes, node); err != nil {
			return nil, fmt.Errorf("failed to deserialize node: %w", err)
		}

		// Deserialize all_tasks if present
		if len(allTasksBytes) > 0 {
			tasksMsg := &pb.TaskList{}
			if err := proto.Unmarshal(allTasksBytes, tasksMsg); err != nil {
				return nil, fmt.Errorf("failed to deserialize all_tasks: %w", err)
			}
			node.AllTasks = tasksMsg.Tasks
		}

		// Deserialize edits if present
		if len(editsBytes) > 0 {
			editsMsg := &pb.NodeEditList{}
			if err := proto.Unmarshal(editsBytes, editsMsg); err != nil {
				return nil, fmt.Errorf("failed to deserialize edits: %w", err)
			}
			node.Edits = editsMsg.Edits
		}

		// Ensure status from database is used (it's the source of truth)
		node.Status = pb.Status(statusInt)

		workflow.Nodes = append(workflow.Nodes, node)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating node rows: %w", err)
	}

	return &workflow, nil
}

// UpdateWorkflowStatus updates the status of a workflow
func (p *PostgresStateManager) UpdateWorkflowStatus(ctx context.Context, workflowID string, status pb.Status) error {
	_, err := p.pool.Exec(ctx,
		`UPDATE workflows SET status = $1, updated_at = $2 WHERE id = $3`,
		int(status), time.Now(), workflowID)
	if err != nil {
		return fmt.Errorf("failed to update workflow status: %w", err)
	}
	return nil
}

// ListWorkflows returns a list of all workflow IDs
func (p *PostgresStateManager) ListWorkflows(ctx context.Context) ([]string, error) {
	rows, err := p.pool.Query(ctx, `SELECT id FROM workflows`)
	if err != nil {
		return nil, fmt.Errorf("failed to query workflows: %w", err)
	}
	defer rows.Close()

	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("failed to scan workflow ID: %w", err)
		}
		ids = append(ids, id)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating workflow rows: %w", err)
	}

	return ids, nil
}

// CreateNode persists a new node to the database
func (p *PostgresStateManager) CreateNode(ctx context.Context, workflowID string, node *pb.Node) error {
	tx, err := p.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	if err := p.createNodeTx(ctx, tx, workflowID, node); err != nil {
		return err
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// GetNode retrieves a node from the database
func (p *PostgresStateManager) GetNode(ctx context.Context, workflowID, nodeID string) (*pb.Node, error) {
	var statusInt int
	var nodeBytes, allTasksBytes, editsBytes []byte
	err := p.pool.QueryRow(ctx,
		`SELECT status, node, all_tasks, edits FROM nodes 
         WHERE workflow_id = $1 AND id = $2`,
		workflowID, nodeID).Scan(&statusInt, &nodeBytes, &allTasksBytes, &editsBytes)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("node not found: %s", nodeID)
		}
		return nil, fmt.Errorf("failed to query node: %w", err)
	}

	// Deserialize node protobuf
	node := &pb.Node{}
	if err := proto.Unmarshal(nodeBytes, node); err != nil {
		return nil, fmt.Errorf("failed to deserialize node: %w", err)
	}

	// Deserialize all_tasks if present
	if len(allTasksBytes) > 0 {
		tasksMsg := &pb.TaskList{}
		if err := proto.Unmarshal(allTasksBytes, tasksMsg); err != nil {
			return nil, fmt.Errorf("failed to deserialize all_tasks: %w", err)
		}
		node.AllTasks = tasksMsg.Tasks
	}

	// Deserialize edits if present
	if len(editsBytes) > 0 {
		editsMsg := &pb.NodeEditList{}
		if err := proto.Unmarshal(editsBytes, editsMsg); err != nil {
			return nil, fmt.Errorf("failed to deserialize edits: %w", err)
		}
		node.Edits = editsMsg.Edits
	}

	// Ensure status from database is used (it's the source of truth)
	node.Status = pb.Status(statusInt)

	return node, nil
}

// UpdateNode updates an existing node in the database
func (p *PostgresStateManager) UpdateNode(ctx context.Context, workflowID string, node *pb.Node) error {
	tx, err := p.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	// Serialize the node protobuf
	nodeBytes, err := proto.Marshal(node)
	if err != nil {
		return fmt.Errorf("failed to serialize node: %w", err)
	}

	// Extract all_tasks and edits for separate storage
	var allTasksBytes, editsBytes []byte
	if len(node.AllTasks) > 0 {
		// Create a temporary message to hold just the tasks
		tasksMsg := &pb.TaskList{
			Tasks: node.AllTasks,
		}
		allTasksBytes, err = proto.Marshal(tasksMsg)
		if err != nil {
			return fmt.Errorf("failed to serialize all_tasks: %w", err)
		}
	}

	if len(node.Edits) > 0 {
		editsMsg := &pb.NodeEditList{
			Edits: node.Edits,
		}
		editsBytes, err = proto.Marshal(editsMsg)
		if err != nil {
			return fmt.Errorf("failed to serialize edits: %w", err)
		}
	}

	// Update node record
	result, err := tx.Exec(ctx,
		`UPDATE nodes SET status = $1, node = $2, all_tasks = $3, edits = $4, updated_at = $5
         WHERE workflow_id = $6 AND id = $7`,
		int(node.Status), nodeBytes, allTasksBytes, editsBytes, time.Now(), workflowID, node.NodeId)
	if err != nil {
		return fmt.Errorf("failed to update node: %w", err)
	}

	rowsAffected := result.RowsAffected()
	if rowsAffected == 0 {
		return fmt.Errorf("node not found: %s", node.NodeId)
	}

	// Update node edges - first delete existing edges
	_, err = tx.Exec(ctx,
		`DELETE FROM node_edges WHERE workflow_id = $1 AND (parent_node_id = $2 OR child_node_id = $2)`,
		workflowID, node.NodeId)
	if err != nil {
		return fmt.Errorf("failed to delete existing edges: %w", err)
	}

	// Insert new parent edges
	for _, parentID := range node.ParentIds {
		_, err = tx.Exec(ctx,
			`INSERT INTO node_edges (workflow_id, parent_node_id, child_node_id) 
             VALUES ($1, $2, $3)`,
			workflowID, parentID, node.NodeId)
		if err != nil {
			return fmt.Errorf("failed to insert parent edge: %w", err)
		}
	}

	// Insert new child edges
	for _, childID := range node.ChildIds {
		_, err = tx.Exec(ctx,
			`INSERT INTO node_edges (workflow_id, parent_node_id, child_node_id) 
             VALUES ($1, $2, $3)`,
			workflowID, node.NodeId, childID)
		if err != nil {
			return fmt.Errorf("failed to insert child edge: %w", err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// ApplyNodeEdits applies a set of node edits to the workflow
func (p *PostgresStateManager) ApplyNodeEdits(ctx context.Context, workflowID string, edits []*pb.NodeEdit) error {
	tx, err := p.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	for _, edit := range edits {
		switch edit.Type {
		case pb.NodeEdit_INSERT:
			// Insert a new node
			if edit.Node == nil {
				return fmt.Errorf("node is required for INSERT edit")
			}
			if err := p.createNodeTx(ctx, tx, workflowID, edit.Node); err != nil {
				return fmt.Errorf("failed to apply INSERT edit: %w", err)
			}

		case pb.NodeEdit_UPDATE:
			// Update an existing node
			if edit.Node == nil {
				return fmt.Errorf("node is required for UPDATE edit")
			}

			// Serialize the node protobuf
			nodeBytes, err := proto.Marshal(edit.Node)
			if err != nil {
				return fmt.Errorf("failed to serialize node for UPDATE: %w", err)
			}

			// Extract all_tasks and edits for separate storage
			var allTasksBytes, editsBytes []byte
			if len(edit.Node.AllTasks) > 0 {
				tasksMsg := &pb.TaskList{
					Tasks: edit.Node.AllTasks,
				}
				allTasksBytes, err = proto.Marshal(tasksMsg)
				if err != nil {
					return fmt.Errorf("failed to serialize all_tasks for UPDATE: %w", err)
				}
			}

			if len(edit.Node.Edits) > 0 {
				editsMsg := &pb.NodeEditList{
					Edits: edit.Node.Edits,
				}
				editsBytes, err = proto.Marshal(editsMsg)
				if err != nil {
					return fmt.Errorf("failed to serialize edits for UPDATE: %w", err)
				}
			}

			// Update node record
			result, err := tx.Exec(ctx,
				`UPDATE nodes SET status = $1, node = $2, all_tasks = $3, edits = $4, updated_at = $5
                 WHERE workflow_id = $6 AND id = $7`,
				int(edit.Node.Status), nodeBytes, allTasksBytes, editsBytes, time.Now(), workflowID, edit.Node.NodeId)
			if err != nil {
				return fmt.Errorf("failed to apply UPDATE edit: %w", err)
			}

			rowsAffected := result.RowsAffected()
			if rowsAffected == 0 {
				return fmt.Errorf("node not found for UPDATE: %s", edit.Node.NodeId)
			}

			// Update node edges - first delete existing edges
			_, err = tx.Exec(ctx,
				`DELETE FROM node_edges WHERE workflow_id = $1 AND (parent_node_id = $2 OR child_node_id = $2)`,
				workflowID, edit.Node.NodeId)
			if err != nil {
				return fmt.Errorf("failed to delete existing edges for UPDATE: %w", err)
			}

			// Insert new parent edges
			for _, parentID := range edit.Node.ParentIds {
				_, err = tx.Exec(ctx,
					`INSERT INTO node_edges (workflow_id, parent_node_id, child_node_id) 
                     VALUES ($1, $2, $3)`,
					workflowID, parentID, edit.Node.NodeId)
				if err != nil {
					return fmt.Errorf("failed to insert parent edge for UPDATE: %w", err)
				}
			}

			// Insert new child edges
			for _, childID := range edit.Node.ChildIds {
				_, err = tx.Exec(ctx,
					`INSERT INTO node_edges (workflow_id, parent_node_id, child_node_id) 
                     VALUES ($1, $2, $3)`,
					workflowID, edit.Node.NodeId, childID)
				if err != nil {
					return fmt.Errorf("failed to insert child edge for UPDATE: %w", err)
				}
			}

		case pb.NodeEdit_DELETE:
			// Delete an existing node
			if edit.Node == nil || edit.Node.NodeId == "" {
				return fmt.Errorf("node ID is required for DELETE edit")
			}

			// Delete node edges first (due to foreign key constraints)
			_, err = tx.Exec(ctx,
				`DELETE FROM node_edges WHERE workflow_id = $1 AND (parent_node_id = $2 OR child_node_id = $2)`,
				workflowID, edit.Node.NodeId)
			if err != nil {
				return fmt.Errorf("failed to delete edges for DELETE edit: %w", err)
			}

			// Delete node record
			result, err := tx.Exec(ctx,
				`DELETE FROM nodes WHERE workflow_id = $1 AND id = $2`,
				workflowID, edit.Node.NodeId)
			if err != nil {
				return fmt.Errorf("failed to apply DELETE edit: %w", err)
			}

			rowsAffected := result.RowsAffected()
			if rowsAffected == 0 {
				return fmt.Errorf("node not found for DELETE: %s", edit.Node.NodeId)
			}

		default:
			return fmt.Errorf("unknown edit type: %v", edit.Type)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// FindReadyNodes finds nodes that are ready to be executed
// A node is ready when:
// 1. Its status is PENDING
// 2. All its parent nodes have status PASS
func (p *PostgresStateManager) FindReadyNodes(ctx context.Context) ([]*pb.Node, error) {
	// This query finds nodes that:
	// 1. Have status = PENDING
	// 2. Either have no parents, or all parents have status = PASS
	rows, err := p.pool.Query(ctx, `
		SELECT n.workflow_id, n.id, n.status, n.node, n.all_tasks, n.edits
		FROM nodes n
		WHERE n.status = $1
		AND NOT EXISTS (
			-- Find any parent that doesn't have status = PASS
			SELECT 1 FROM node_edges e
			JOIN nodes parent ON e.parent_node_id = parent.id AND e.workflow_id = parent.workflow_id
			WHERE e.child_node_id = n.id
			AND e.workflow_id = n.workflow_id
			AND parent.status != $2
		)
	`, int(pb.Status_UNKOWN), int(pb.Status_PASS))
	if err != nil {
		return nil, fmt.Errorf("failed to query ready nodes: %w", err)
	}
	defer rows.Close()

	var readyNodes []*pb.Node
	for rows.Next() {
		var workflowID, nodeID string
		var statusInt int
		var nodeBytes, allTasksBytes, editsBytes []byte
		if err := rows.Scan(&workflowID, &nodeID, &statusInt, &nodeBytes, &allTasksBytes, &editsBytes); err != nil {
			return nil, fmt.Errorf("failed to scan ready node row: %w", err)
		}

		// Deserialize node protobuf
		node := &pb.Node{}
		if err := proto.Unmarshal(nodeBytes, node); err != nil {
			return nil, fmt.Errorf("failed to deserialize ready node: %w", err)
		}

		// Deserialize all_tasks if present
		if len(allTasksBytes) > 0 {
			tasksMsg := &pb.TaskList{}
			if err := proto.Unmarshal(allTasksBytes, tasksMsg); err != nil {
				return nil, fmt.Errorf("failed to deserialize all_tasks for ready node: %w", err)
			}
			node.AllTasks = tasksMsg.Tasks
		}

		// Deserialize edits if present
		if len(editsBytes) > 0 {
			editsMsg := &pb.NodeEditList{}
			if err := proto.Unmarshal(editsBytes, editsMsg); err != nil {
				return nil, fmt.Errorf("failed to deserialize edits for ready node: %w", err)
			}
			node.Edits = editsMsg.Edits
		}

		// Ensure status from database is used (it's the source of truth)
		node.Status = pb.Status(statusInt)

		readyNodes = append(readyNodes, node)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating ready node rows: %w", err)
	}

	return readyNodes, nil
}
