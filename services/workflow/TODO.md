# Workflow Service TODO

*   [x] **(High Priority - CONV-003 Part 1)** Adapt core engine for AI agent workflows.
    *   [x] Define generic task execution interface.
    *   [x] Implement Protobuf communication layer (client side for talking to nodes).
*   [x] High-level design for workflow service
*   [x] Low-level design for workflow service state machine & storage, & interface with node service
*   [ ] Implement workflow state persistence mechanism.
*   [ ] Develop initial scheduler logic (simple capability matching).
*   [ ] Build basic API for defining and launching workflows.
*   [ ] Implement event emission for key state changes.
*   [ ] Set up basic monitoring dashboard/interface.
*   [ ] Integrate with [[protos/TODO.md|Protobuf definitions]] once stable.
---

### Detailed Implementation Plan (Milestone Commits)

#### 1. Persistence Layer
- [ ] Define Go structs/interfaces for `StateManager` abstraction.
- [ ] Implement PostgreSQL connection setup and configuration.
- [ ] Implement `CreateWorkflow` method: persist initial workflow and nodes.
- [ ] Implement `GetWorkflow` method: retrieve and deserialize workflow.
- [ ] Implement `UpdateWorkflowStatus` method.
- [ ] Implement `CreateNode` and `UpdateNode` methods with protobuf serialization.
- [ ] Implement transactional `ApplyNodeEdits` method.
- [ ] Write unit tests for `StateManager` methods.

#### 2. Scheduler & Orchestration Engine
- [ ] Implement polling loop to scan active workflows.
- [ ] Implement query to find ready nodes (all parents `PASS`, status `PENDING`).
- [ ] Implement simple capability matching logic.
- [ ] Implement dispatch queue respecting DAG dependencies.
- [ ] Update node status to `RUNNING` upon dispatch.
- [ ] Write tests for scheduling logic.

#### 3. API Layer (gRPC `WorkflowService`)
- [ ] Implement `CreateWorkflow` RPC handler.
- [ ] Implement `GetWorkflow` RPC handler.
- [ ] Implement `ListWorkflows` RPC handler.
- [ ] Implement `UpdateWorkflow` RPC handler.
- [ ] Implement `GetNode` and `UpdateNode` RPC handlers.
- [ ] Add input validation and error handling.
- [ ] Write API layer tests with mocked `StateManager`.

#### 4. Node Dispatch & Response Handling
- [ ] Implement gRPC client for `NodeService`.
- [ ] Construct `ExecuteNodeRequest` with node and context.
- [ ] Send request and handle `ExecuteNodeResponse`.
- [ ] Update node status and results on success.
- [ ] Handle agent errors (`TASK_ERROR`) and infra errors (`INFRA_ERROR`).
- [ ] Apply any returned `NodeEdit`s transactionally.
- [ ] Log dispatch and response details.

#### 5. Event Emission
- [ ] Define event types and payloads.
- [ ] Implement simple event logging (stdout or DB).
- [ ] Emit events on workflow/node creation, dispatch, completion, edits.
- [ ] Plan integration with event bus (future).

#### 6. Monitoring & Observability
- [ ] Add Prometheus metrics counters/gauges (active workflows, node statuses, errors).
- [ ] Add execution time histograms.
- [ ] Implement structured logging for API, scheduler, dispatch.
- [ ] Plan basic dashboard UI (future).

#### 7. Dynamic Workflow Editing
- [ ] Implement parsing and validation of `NodeEdit` messages.
- [ ] Support `INSERT`, `UPDATE`, `DELETE` edit types.
- [ ] Apply edits within DB transactions.
- [ ] Recompute scheduling after edits.
- [ ] Add tests for edit scenarios.

#### 8. Testing & Validation
- [ ] Write integration tests covering workflow lifecycle.
- [ ] Simulate node execution with mock `NodeService`.
- [ ] Test error handling and retries.
- [ ] Validate persistence and recovery on restart.
- [ ] Document example workflows and usage.

---

This plan enables incremental, testable commits aligned with the architecture and design documents.
