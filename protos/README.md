# Protocol Buffers

**Purpose:** This directory contains the Protocol Buffer (`.proto`) definitions used for all strictly typed communication between AI Society agents and services.

**Rationale:** Using Protobufs ensures:
*   **Type Safety:** Reduces errors from ambiguous or malformed messages.
*   **Interoperability:** Allows agents/services potentially written in different languages/frameworks to communicate reliably.
*   **Structure:** Enforces a clear schema for data exchange (tasks, results, status updates, events, configuration).
*   **Efficiency:** Provides a compact binary wire format.

**Core Components:**
*   Task definitions and results structures.
*   Agent/Node configuration and status messages.
*   Workflow state updates and events.
*   Potentially messages for governance proposals, votes, knowledge submission.

---

## `workflow_node.proto` Overview

Defines the core data structures for representing and managing workflow graphs:

- **Node**: Represents a node in the workflow graph, including:
  - `node_id`, `description`
  - Parent and child node references
  - Assigned `AgentIdentity`
  - The `Task` assigned to this node
  - All tasks in the workflow (`all_tasks`)
  - Current `Status`
  - List of `NodeEdit` changes applied to the graph

- **AgentIdentity**: Identifies an agent, including:
  - `agent_id`
  - `role` (Planner, Worker, Supervisor, etc.)
  - `model_type` (Claude-3, GPT-4, etc.)

- **Task**: Represents a task with:
  - Unique `id` and `goal`
  - Dependencies (`dependency_ids`)
  - Nested `subtasks`
  - List of `Result` objects, each with:
    - `status`
    - `summary`
    - `output`
    - `artifacts` (named outputs/files)

- **Status**: Enum indicating task/node status (PASS, FAIL, SKIPPED, ERROR, etc.)

- **NodeEdit**: Captures graph modifications (INSERT, DELETE, UPDATE) with timestamp and description.

- **NodeStatus**: (Placeholder) Tracks node execution state, progress updates, and timestamps.

---

**Relation to Architecture:** This is a foundational element of the chosen architecture, enabling communication between the [[services/workflow/README.md|Workflow Service]] and the various [[services/node/README.md|Node Executors]]. See [[services/workflow/Architecture.md]].

**Current Tasks:** See [[protos/TODO.md]].
