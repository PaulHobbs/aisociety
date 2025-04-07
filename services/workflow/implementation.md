# Workflow Service - Low-Level Design (LLD)

---

## 1. Overview

This LLD specifies the internal components, data flows, persistence, and interactions of the Workflow Service, focusing on:

- Workflow state machine & storage
- Interface with Node Service
- Scheduler logic
- API endpoints
- Event emission
- Monitoring hooks

---

## 2. Core Data Models

### 2.1 Workflow

- **ID:** UUID (generated on creation)
- **Metadata:** Name, description, creator info
- **Nodes:** Map of `node_id` → `Node` (protobuf serialized)
- **Status:** Aggregate status (computed from node states)
- **Created/Updated timestamps**

**Storage:**  
- PostgreSQL table `workflows`  
- Columns: `workflow_id (PK)`, `metadata (JSONB)`, `nodes (BYTEA)`, `status`, `created_at`, `updated_at`

### 2.2 Node (protobuf `Node` message)

- Serialized and stored as BYTEA within the workflow record
- Key fields:
  - `node_id`, `parent_ids`, `child_ids`
  - `agent` (type, model, role)
  - `assigned_task` (goal, dependencies, results)
  - `status` (enum: PENDING, RUNNING, PASS, FAIL, etc.)
  - `edits` (list of `NodeEdit`)
  - `is_final` (bool)

### 2.3 Task (protobuf `Task` message)

- Embedded within `Node.assigned_task`
- Contains goal, dependencies, results, subtasks

---

## 3. Workflow State Machine

### 3.1 Node States

- **PENDING:** Waiting for dependencies
- **RUNNING:** Dispatched to Node Service
- **PASS:** Completed successfully
- **FAIL:** Completed with failure
- **TASK_ERROR, INFRA_ERROR, TIMEOUT, CRASH:** Error subtypes
- **SKIPPED, FILTERED:** Not executed due to planning decisions

### 3.2 Workflow Lifecycle

1. **Created:** Initial nodes persisted, all in PENDING
2. **Scheduled:** Scheduler identifies ready nodes (all parents PASS)
3. **Dispatched:** Node status set to RUNNING, sent to Node Service
4. **Completed:** Node returns PASS/FAIL, results saved
5. **Edited:** Node may return `NodeEdit`s, modifying graph/tasks
6. **Progressed:** New nodes become ready, repeat scheduling
7. **Terminated:** All terminal nodes PASS or unrecoverable error

---

## 4. Persistence Layer

### 4.1 Database Schema

- **Table:** `workflows`
- **Columns:**
  - `workflow_id` (UUID, PK)
  - `metadata` (JSONB)
  - `nodes` (BYTEA, serialized protobuf map or list)
  - `status` (string)
  - `created_at`, `updated_at` (timestamps)

### 4.2 Storage Strategy

- Entire workflow graph stored as a protobuf-serialized blob for atomic updates
- Optionally, maintain a separate `nodes` table for indexing/querying individual nodes (future optimization)
- Use transactions to ensure consistency during updates (node status, edits)

### 4.3 StateManager Component

- Handles:
  - Serialization/deserialization of protobuf `Node`s
  - Atomic updates to workflow record
  - Querying workflows by ID, status
  - Updating node status/results
  - Applying `NodeEdit`s

---

## 5. Scheduler & Orchestration Engine

### 5.1 Responsibilities

- Periodically scan active workflows
- Identify nodes where:
  - All `parent_ids` have status `PASS`
  - Node status is `PENDING`
- Dispatch ready nodes to Node Service
- Update node status to `RUNNING`
- Handle responses, update status/results
- Apply any `NodeEdit`s returned
- Re-evaluate workflow graph after edits

### 5.2 Scheduling Algorithm (Initial)

- **Simple capability matching:**
  - Check `Node.agent.role` and `Node.assigned_task.goal`
  - Optionally, filter by agent availability or load (future)
- **Dispatch order:**
  - Topological order respecting dependencies
  - Parallel dispatch of independent nodes

### 5.3 Dispatch Flow

1. Build `ExecuteNodeRequest`:
   - `workflow_id`, `node_id`
   - `node` (full protobuf)
   - `upstream_nodes` (parents)
   - `downstream_nodes` (children)
2. Call `NodeService.ExecuteNode()`
3. On response:
   - Update node status, results
   - Persist updated node
   - Apply any `NodeEdit`s
   - Reschedule as needed

---

## 6. Interface with Node Service

### 6.1 gRPC Client

- Use generated protobuf client for `NodeService`
- Call `ExecuteNode(ExecuteNodeRequest)`

### 6.2 Request Construction

- Include:
  - Full `Node` protobuf
  - Context from upstream/downstream nodes
- Pass `assigned_task` goal and dependencies
- Provide agent info for execution

### 6.3 Response Handling

- On success:
  - Update node status to `PASS`
  - Save results in `assigned_task.results`
- On failure:
  - Update node status to `FAIL` or error subtype
  - Save error details
- On edits:
  - Apply `NodeEdit`s to workflow graph
  - Persist changes

---

## 7. API Layer

### 7.1 Endpoints (gRPC `WorkflowService`)

- `CreateWorkflow(CreateWorkflowRequest)`
- `GetWorkflow(GetWorkflowRequest)`
- `ListWorkflows(ListWorkflowsRequest)`
- `UpdateWorkflow(UpdateWorkflowRequest)`
- `GetNode(GetNodeRequest)`
- `UpdateNode(UpdateNodeRequest)`

### 7.2 API Flow

- **CreateWorkflow:**
  - Validate input nodes/tasks
  - Assign workflow ID
  - Persist initial state
- **GetWorkflow:**
  - Fetch workflow by ID
  - Deserialize nodes
- **UpdateWorkflow:**
  - Apply updates (e.g., edits)
  - Persist new state
- **Node APIs:**
  - Fetch/update individual node status/results

---

## 8. Event Emission

### 8.1 Key Events

- Workflow created
- Node scheduled
- Node dispatched
- Node completed (success/failure)
- Workflow completed/failed
- Node edits applied

### 8.2 Mechanism

- Initially, simple event logs to stdout or database table
- Future: integrate with Event Bus (Kafka, NATS, etc.)
- Emit events asynchronously to avoid blocking core flow

---

## 9. Monitoring & Observability

### 9.1 Metrics

- Number of active workflows
- Node status counts (pending, running, pass, fail)
- Execution times per node/workflow
- Error rates

### 9.2 Logs

- Detailed execution logs per node (stored in DB)
- API request/response logs
- Scheduler activity logs

### 9.3 Dashboard (Future)

- Basic UI to view workflows, node graphs, statuses
- Drill-down into node results and logs

---

## 10. Error Handling

- **Agent errors:** Mark node as `TASK_ERROR`, save error output
- **Infrastructure errors:** Mark as `INFRA_ERROR`
- **Timeouts:** Enforced via `ExecutionOptions.timeout`
- **Retries:** Controlled by `ExecutionOptions.retry_options`
- **Unrecoverable errors:** Mark workflow as failed, emit event

---

## 11. Dynamic Workflow Editing

- Process `NodeEdit` messages returned by nodes
- Types:
  - **INSERT:** Add new nodes/tasks
  - **UPDATE:** Modify existing nodes/tasks
  - **DELETE:** Remove nodes/tasks
- Apply edits transactionally
- Recompute scheduling after edits

---

## 12. Component Diagram (Textual)

```
[Client] 
   |
   v
[WorkflowService API Layer]
   |
   v
[StateManager] <--> [PostgreSQL]
   |
   v
[Scheduler/Orchestration Engine]
   |
   v
[gRPC Client]
   |
   v
[NodeService (Executes Node)]
```

---

## 13. Summary of TODO Coverage

| TODO Item                                               | LLD Section(s)                     |
|---------------------------------------------------------|-----------------------------------|
| Low-level design for state machine & storage            | 2, 3, 4                           |
| Implement workflow state persistence mechanism           | 4                                 |
| Develop initial scheduler logic                          | 5                                 |
| Build basic API for defining and launching workflows     | 7                                 |
| Implement event emission for key state changes           | 8                                 |
| Set up basic monitoring dashboard/interface              | 9                                 |
| Integrate with Protobuf definitions                      | Throughout (esp. 2, 6, 7)         |