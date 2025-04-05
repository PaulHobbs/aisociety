# Protos TODO

*   [x] **(High Priority - CONV-001)** Design initial `.proto` definitions:
    *   `TaskDefinition`: **Partially complete** — see `Task` message in `workflow_node.proto`.
    *   `TaskResult`: **Partially complete** — see `Task.Result` nested message.
    *   `NodeStatus`: **Partially complete** — see `NodeStatus` message and `Status` enum.
    *   `AgentIdentity`: **Initial version complete** — see `AgentIdentity` message.
    *   `WorkflowEvent`: **Not yet defined** — design event messages for task assignment, completion, failure, state changes.

*   [ ] Define schemas for:
    *   Delegation requests and context passing between agents/services.
    *   Submitting proposals to governance or knowledge to [[books/README.md|Books]].
    *   Additional workflow events and notifications.

*   [ ] Set up tooling for compiling `.proto` files for all target languages/services.

*   [ ] Establish versioning strategy for schemas (backward compatibility, deprecation, migration).
