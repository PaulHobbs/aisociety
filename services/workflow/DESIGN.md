# High-Level Design: Workflow Service

## 1. Introduction: Orchestrating the AI Society

**Motivation:** As the AI Society undertakes increasingly complex objectives, coordinating the actions of diverse agents becomes crucial. We need a robust system to define, execute, monitor, and reproduce multi-step processes involving potentially many specialized agents. Manually managing these interactions is inefficient and error-prone.

**Purpose:** The Workflow Service acts as the central nervous system for task execution within the AI Society. It provides the mechanisms to:
*   Define complex tasks as structured workflows.
*   Orchestrate the execution of individual steps within these workflows by assigning them to appropriate agents (via the Node Service).
*   Reliably track the state and progress of workflows and their constituent parts.
*   Capture detailed execution history and data flow for analysis, debugging, and reproducibility.

**Foundation:** This service adapts proven concepts from other orchestration systems, tailored for the unique needs of coordinating AI agents.

## 2. Core Concepts: Workflows, Nodes, and Tasks

At the heart of the service are a few key concepts:

*   **Workflows as Graphs:** A workflow represents a complete process designed to achieve a larger goal. It's modeled as a Directed Acyclic Graph (DAG). This structure allows us to define dependencies and parallel execution paths clearly. Each workflow has a unique ID, metadata (name, description), and an overall status.

*   **Nodes: The Units of Execution:** Nodes are the vertices in the workflow DAG. A Node represents a specific step or unit of work within the workflow. Crucially, each Node is associated with an `Agent` (defined in `workflow_node.proto`) responsible for performing the work. Key attributes of a Node include:
    *   `node_id`: Unique identifier within the workflow.
    *   `agent`: Specifies the agent (type, model, role) assigned to this node.
    *   `status`: Tracks the execution state (e.g., `PENDING`, `RUNNING`, `PASS`, `FAIL`).
    *   `parent_ids` / `child_ids`: Define the DAG structure and dependencies.
    *   `execution_options`: Specifies parameters like timeouts and retries.

*   **Tasks: Defining the Work:** While a Node represents the *execution step*, a `Task` defines the *actual work* to be done or the goal to be achieved. Tasks exist independently of the execution graph structure initially and can be seen as the objectives the workflow aims to fulfill. Key attributes of a Task include:
    *   `id`: Unique identifier for the task objective.
    *   `goal`: A description of what needs to be accomplished.
    *   `dependency_ids`: Specifies dependencies between tasks themselves (which might influence workflow graph creation).
    *   `results`: Stores the outcomes of attempting the task (status, summary, output, artifacts).
    *   `subtasks`: Allows for hierarchical task decomposition.

*   **The Node-Task Relationship:** This is a critical distinction. A `Node` is an entity *in the execution graph*, assigned to an `Agent`. A `Task` is a description of *work*. The relationship manifests in several ways:
    1.  **Assignment:** Each executable `Node` is typically assigned *one* primary `Task` to fulfill. This is stored in the `Node.assigned_task` field. The agent associated with the Node performs the work described in this `assigned_task`.
    2.  **Context:** A `Node` might receive the context of *all* tasks defined for the workflow (`Node.all_tasks`). This is particularly relevant for planning or scheduling nodes that need a global view.
    3.  **Manipulation:** Crucially, Nodes (especially planning/scheduling ones) can *modify* the workflow itself or the tasks within it. They do this by outputting `NodeEdit` messages. These edits might involve changing a `Task`'s goal, adding new `Task`s, creating new `Node`s in the graph, or adjusting dependencies. The `WorkflowService` processes these edits to dynamically adapt the workflow.

## 3. Architecture and Execution Flow

**Service Responsibilities:** The `WorkflowService` (implemented in Go within `services/workflow`) exposes a gRPC API (`WorkflowService` definition in `workflow_node.proto`). Its primary responsibilities include:
*   Managing the lifecycle of workflows (creation, updates, retrieval).
*   Orchestrating the execution flow based on the DAG dependencies.
*   Persisting the state of workflows, nodes, and tasks using a `StateManager` component interacting with PostgreSQL.
*   Interpreting and applying `NodeEdit`s produced by executing nodes within database transactions.
*   Communicating with the `NodeService` via a gRPC client to dispatch work.
*   Providing a gRPC server implementation (`type server struct { pb.UnimplementedWorkflowServiceServer ... }`).

**Decoupled Execution:** A key architectural principle is the separation of concerns between orchestration (`WorkflowService`) and execution (`NodeService`). The `WorkflowService` determines *what* needs to run and *when*, while the `NodeService` handles the specifics of *how* a given node (and its assigned agent/task) is executed. This promotes modularity and allows different execution backends.

**Execution Lifecycle Narrative:**
1.  **Initiation:** A client (internal service or CLI) requests workflow creation via the gRPC API, providing the initial set of nodes and/or tasks.
2.  **Persistence:** The `WorkflowService` validates the request and uses the `StateManager` to persist the initial workflow structure (`workflows` table) and node states (`nodes` table, potentially storing the `pb.Node` proto as `BYTEA`) in the PostgreSQL database (defined in `schema.sql`). Nodes typically start in a `PENDING` status.
3.  **Scheduling Loop:** The Orchestration Engine component runs a continuous loop (e.g., polling the database every few seconds). In each iteration, it queries the `StateManager` for `PENDING` nodes whose parent nodes (tracked via dependencies in the `nodes` table or within the serialized `pb.Node`) have all reached a `PASS` status. On service startup, this loop also handles recovering workflows that were `RUNNING`.
4.  **Dispatch:** For each ready node, the Engine constructs an `ExecuteNodeRequest` (including the `Node` definition, its `assigned_task`, and potentially context from upstream/downstream nodes) and sends it to the `NodeService` via a gRPC client. The node's status is updated to `RUNNING`.
5.  **Execution:** The `NodeService` receives the request, identifies the correct agent based on `Node.agent`, prepares the necessary input/prompt (using `Node.assigned_task.goal` and potentially upstream results), invokes the agent, and awaits the result.
6.  **Result Handling:** The `NodeService` packages the outcome (the complete updated `pb.Node` including status, results, artifacts, and any generated `pb.NodeEdit`s) into an `ExecuteNodeResponse` and returns it to the `WorkflowService`. If the `NodeService` encounters an internal error *preventing* execution (e.g., cannot contact the agent), it should return a gRPC error. If the *agent* fails, the `NodeService` should update the `Node.status` to `TASK_ERROR` and return the updated node in the response, *not* a gRPC error.
7.  **State Update & Edits:** The `WorkflowService` receives the response.
    *   **On Success:** It uses the `StateManager` to update the corresponding `Node` record in the database with the received `pb.Node` data (serialized). If `NodeEdit`s are present in the response's `Node.edits` field, the Orchestration Engine applies these edits *within the same database transaction* used to update the node state. Applying edits involves potentially inserting new nodes, updating existing ones, or changing dependencies based on the `NodeEdit` messages. Clear logging should indicate which edits were applied. Conflicting edits might require a defined resolution strategy (e.g., last write wins, or failing the transaction if atomicity is critical).
    *   **On gRPC Error from NodeService:** The `WorkflowService` should update the node's status to `INFRA_ERROR` via the `StateManager`, potentially retrying based on `ExecutionOptions`.
8.  **Progression:** After a node completes (successfully or with `TASK_ERROR`/`INFRA_ERROR`) and its state (and any edits) are persisted, the Engine's next scheduling loop iteration will naturally re-evaluate dependencies and potentially identify new nodes that are ready for dispatch.
9.  **Completion/Termination:** The workflow completes when all terminal nodes reach a final state or if an unrecoverable error occurs.

**State Management & Observability:** Reliable state persistence is handled by a `StateManager` component (likely a Go interface implemented by a struct interacting with the `database/sql` package and a PostgreSQL driver like `pgx`).
    ```go
    // Potential StateManager Interface (Illustrative)
    type StateManager interface {
        CreateWorkflow(ctx context.Context, workflow *pb.Workflow) error
        GetWorkflow(ctx context.Context, workflowID string) (*pb.Workflow, error)

        CreateNode(ctx context.Context, node *pb.Node) error
        GetNode(ctx context.Context, workflowID, nodeID string) (*pb.Node, error)
        UpdateNode(ctx context.Context, node *pb.Node) error // Used for status, results, etc.
        ApplyNodeEdits(ctx context.Context, workflowID string, edits []*pb.NodeEdit) error // Needs transactional logic

        // Query for nodes ready to run (implementation detail)
        FindReadyNodes(ctx context.Context) ([]*pb.Node, error)
        // ... other necessary methods
    }
    ```
    This interface abstracts the database operations. The implementation will handle serializing `pb.Node` (and potentially `pb.Workflow`, `pb.Task`) into `BYTEA` columns using `proto.Marshal` and deserializing using `proto.Unmarshal`. High-fidelity observability is achieved by storing rich data within the serialized `Node` Protobuf message itself (e.g., in `Node.assigned_task.results`, `Node.edits`, status history).

## 4. Advanced Workflow Patterns with `Node.Edits`

The `Node.Edits` mechanism enables dynamic and intelligent workflows. Here are illustrative examples:

*   **Configuration Nodes:** Imagine a workflow starting with a "Load Config" node. This node might read a configuration file or call an external service. Based on the retrieved configuration, it generates `NodeEdit` messages of type `UPDATE` to populate the `assigned_task` fields (e.g., setting specific parameters or goals) of downstream nodes before they execute. It could even use `INSERT` edits to add entirely new nodes/tasks based on the configuration.

*   **Planning Nodes:** A workflow might have a high-level goal like "Write a report on topic X". A "Planner" node could be assigned this initial task. This node's agent would analyze the goal, perhaps consult a knowledge base, and then generate a series of `NodeEdit`s. These edits would `INSERT` new nodes into the graph representing sub-steps (e.g., "Research Topic X", "Draft Outline", "Write Introduction", "Write Body", "Write Conclusion", "Review Report"), establishing dependencies between them. It might also `UPDATE` the original task description or `DELETE` placeholder nodes.

*   **Scheduling/Decomposition Nodes:** Consider a "Process Dataset" node assigned a large task. A "Scheduler" agent associated with this node could analyze the dataset size and available worker agents. It might then decompose the `assigned_task` by adding detailed `subtasks` within the `Node.assigned_task.subtasks` field. Alternatively, it could use `NodeEdit`s of type `INSERT` to add multiple parallel "Worker" nodes to the graph, each assigned a specific chunk of the dataset, effectively distributing the load.

These examples show how `Node.Edits` allow nodes to actively shape the workflow as it executes, enabling adaptation, planning, and dynamic resource allocation.

## 5. API and Clients

*   **API:** The primary interface is the gRPC `WorkflowService` defined in `protos/workflow_node.proto`. It provides RPCs for creating, retrieving, listing, and updating workflows and their nodes.
*   **Clients:** Initially, the main clients will be other internal AI Society backend services and potentially a command-line interface (CLI) for administrative or development purposes.

## 6. Design Considerations

*   **Communication:** gRPC is chosen for efficient, strongly-typed communication between services.
*   **Persistence:** PostgreSQL provides robust relational storage. Storing core data like `workflow_id`, `node_id`, `status`, and dependency links in standard columns allows for efficient querying by the Orchestration Engine. Storing the full `pb.Node` Protobuf message in a `BYTEA` column offers flexibility for evolving the `Node` structure without immediate schema migrations. Database operations should be performed within transactions where atomicity is required (e.g., updating a node and applying its edits).
*   **Extensibility:** The architecture accommodates diverse agent types via the standardized `NodeService` gRPC interface.
*   **Error Handling:** Clear distinction between agent/task errors (`TASK_ERROR`, handled by updating node status) and infrastructure/communication errors (`INFRA_ERROR`, potentially leading to retries or workflow failure). The `WorkflowService` is responsible for interpreting results/errors from `NodeService` and updating state accordingly via the `StateManager`.
*   **Configuration:** Service configuration (database connection string, NodeService address, polling intervals, etc.) should be managed externally (e.g., environment variables, configuration files loaded at startup).
*   **Concurrency:** The Orchestration Engine might dispatch multiple nodes concurrently. Database transactions and potentially optimistic/pessimistic locking strategies (if needed, though careful state management might suffice initially) should be used to prevent race conditions when updating workflow/node states.
*   **Reliability/Scalability:** For initial use cases, a single instance polling the database is likely sufficient. Future scaling could involve more sophisticated queue-based dispatching or sharding.

## 7. Future Considerations

Areas for future exploration and enhancement include:
*   Implementing more sophisticated error handling and retry strategies.
*   Developing advanced scheduling logic (considering agent capabilities, load, priorities).
*   Defining and implementing comprehensive monitoring, metrics, and alerting (beyond the node-level data).
*   Adding robust API security (authentication/authorization).
*   Optimizing the handling of potentially large data artifacts generated by tasks.
*   Refining the specific Protobuf fields and structure used for capturing detailed event data within the `Node`.
*   Implementing the conceptual Event Bus for broader system notifications.