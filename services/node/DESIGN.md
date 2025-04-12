# Node Service: MCP Tool Invocation Design

## Overview

The Node Service is responsible for executing workflow nodes as assigned by the Workflow Service. It provides a harness for agents to fulfill their assigned tasks, which may include invoking external MCP (Model Context Protocol) tools. This document details how the Node Service discovers and invokes MCP tools as part of agent execution, referencing the workflow architecture and proto definitions.

---

## Architectural Context

The Workflow Service orchestrates execution by dispatching ready nodes to the Node Service via the `ExecuteNode` RPC. Each node includes:
- The assigned agent (`Node.agent`)
- The assigned task (`Node.assigned_task`)
- Execution options (timeouts, retries)
- Context (upstream/downstream nodes, all tasks)

The Node Service is responsible for:
- Interpreting the assigned task and agent
- Preparing the agent harness environment
- Discovering and invoking MCP tools as required by the agent/task
- Collecting results, status, and any workflow edits
- Returning an updated `Node` (with results, status, and edits) in the `ExecuteNodeResponse`

---

## Integration Points

### 1. Task Interpretation and Tool Selection

- The agent harness receives the `Node` and its `assigned_task`.
- The harness inspects the task goal, parameters, and context to determine if an MCP tool invocation is required.
  - For example, a task goal may specify `"Call: knowledge-base-curator.summarize"` or include a tool name in structured parameters.
- The harness queries the Node Service's MCP tool registry to discover available tools and their schemas.

### 2. MCP Tool Invocation

- The agent harness constructs a request for the selected MCP tool, mapping task parameters to the tool's input schema.
- The Node Service invokes the MCP tool via the appropriate adapter (local process, gRPC, HTTP, etc.).
- The tool executes and returns a result or error.

### 3. Result and Edit Handling

- The agent harness processes the tool result, updating the `Node.assigned_task.results` field with:
  - `status` (PASS, FAIL, TASK_ERROR, etc.)
  - `summary` and `output`
  - Any generated artifacts (logs, files, URLs)
- If the agent determines that workflow edits are needed (e.g., inserting new nodes, updating tasks), it populates the `Node.edits` field with `NodeEdit` messages.
- The updated `Node` is returned in the `ExecuteNodeResponse` to the Workflow Service.

---

## Proto Message References

- **ExecuteNodeRequest**
  - `workflow_id`, `node_id`, `node` (to execute), `upstream_nodes`, `downstream_nodes`
- **Node**
  - `agent`, `assigned_task`, `status`, `edits`, `execution_options`
- **Task**
  - `goal`, `results`, `subtasks`
- **NodeEdit**
  - Used for dynamic workflow changes (insert, update, delete nodes/tasks)
- **ExecuteNodeResponse**
  - Returns the updated `Node` with results and edits

---

## Error Handling

- **Agent/Task Errors:** If the MCP tool or agent logic fails, the harness sets `Node.status = TASK_ERROR` and includes error details in `assigned_task.results`.
- **Infrastructure Errors:** If the Node Service cannot execute the node (e.g., tool registry unavailable), it returns a gRPC error. The Workflow Service will update the node status to `INFRA_ERROR` and may retry.
- **Timeouts/Crashes:** The harness respects `Node.execution_options.timeout` and updates status to `TIMEOUT` or `CRASH` as appropriate.
- **Partial Results:** If a tool produces partial output, the harness can record this in the results and set an appropriate status.

---

## Extensibility

- **Adding New Tools:** MCP tools can be registered with the Node Service at startup or dynamically. The registry is exposed to agent harnesses for discovery.
- **Schema Evolution:** Tool input/output schemas are versioned. The Node Service validates requests and responses for compatibility.
- **Adapters:** The Node Service supports pluggable adapters for different tool protocols (gRPC, HTTP, local process).

---

## Security Considerations

- **Authentication:** Only authorized agents and tools can be registered or invoked.
- **Authorization:** The Node Service enforces policies restricting which agents/tasks can invoke which tools.
- **Input Validation:** All tool inputs are validated against schemas to prevent malformed or malicious requests.
- **Auditing:** All tool invocations, results, and edits are logged for traceability.

---

## Example Execution Flow

```
WorkflowService         NodeService/AgentHarness         MCP Tool Adapter         MCP Tool
      |                        |                              |                      |
1. Dispatch node:              |                              |                      |
   ExecuteNodeRequest -------->|                              |                      |
      |                        |                              |                      |
2. Prepare agent harness       |                              |                      |
   with Node, Task, etc.       |                              |                      |
      |                        |                              |                      |
3. Agent determines tool       |                              |                      |
   to invoke (from task)       |                              |                      |
      |                        |---InvokeTool(tool, input)--->|                      |
      |                        |                              |---Call-------------->|
      |                        |                              |<--Result/Error-------|
      |                        |<--Result/Error---------------|                      |
4. Agent updates Node:         |                              |                      |
   - assigned_task.results     |                              |                      |
   - status                    |                              |                      |
   - edits (if any)            |                              |                      |
      |                        |                              |                      |
5. ExecuteNodeResponse <-------|                              |                      |
      |                        |                              |                      |
6. WorkflowService updates     |                              |                      |
   node state, applies edits   |                              |                      |
```

---

## Separation of Concerns

- **Workflow Service:** Orchestrates execution, tracks state, applies edits, persists data.
- **Node Service:** Executes nodes, manages agent harness, invokes MCP tools, reports results and edits.
- **Agent Harness:** Implements agent logic, determines tool usage, processes results, generates edits.
- **MCP Tool Adapter:** Handles protocol-specific communication with external tools.

---

## Implementation Notes

- The agent harness should always report subtask status and results in `Node.assigned_task.results`.
- All edits to the workflow (e.g., new nodes, task updates) must be encoded as `NodeEdit` messages in the response.
- The Node Service should be stateless between executions; all state is passed via proto messages.
- For testing, adapters and tool registries can be mocked.

---

## References

- [protos/workflow_node.proto](../../protos/workflow_node.proto)
- [services/workflow/DESIGN.md](../workflow/DESIGN.md)
- [Node Service Source](./)
- [MCP Tool Examples](../../mcp-tools/)