# Protos TODO

*   [x] **P1** Design initial `.proto` definitions for workflow graphs:
    *   `Task`: ** complete** — see `Task` message in `workflow_node.proto`.
    *   `TaskResult`: ** complete** — see `Task.Result` nested message.
    *   `NodeStatus`: ** complete** — see `NodeStatus` message and `Status` enum.
    *   `AgentIdentity`: ** version complete** — see `AgentIdentity` message.
*   [ ] P1 Set up tooling for compiling `.proto` files for all target languages/services.
*   [ ] P2 `WorkflowEvent`: **Not yet defined** — design event messages for task assignment, completion, failure, state changes.
*   [ ] P2 Define schemas for
    * Submitting proposals to governance or knowledge to [[books/README.md|Books]].
    * Persistent / looping agents who have an external clock outside of a workflow graph
    * Discord-style message passing (can we just use discord directly?)    
