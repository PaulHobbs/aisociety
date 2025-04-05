# Protos TODO

*   [ ] **(High Priority - CONV-001)** Design initial `.proto` definitions for:
    *   `TaskDefinition`: Inputs, description, requirements, ID.
    *   `TaskResult`: Output data (structured), status, logs, originating task ID.
    *   `NodeStatus`: Health, current task, capabilities.
    *   `WorkflowEvent`: Task assignment, completion, failure, state changes.
    *   `AgentIdentity`: Model type, role, unique ID. (Refine Claude's `Identity` message).
*   [ ] Define schemas for delegation requests and context passing.
*   [ ] Define schemas for submitting proposals to governance or knowledge to [[books/README.md|Books]].
*   [ ] Set up tooling for compiling `.proto` files for target languages/services.
*   [ ] Establish versioning strategy for schemas.
