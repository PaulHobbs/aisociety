# Node Executor TODO

**Architectural Note:**
The Node Service is responsible for providing the agent harness (runtime environment) and for executing any tools or external resources required by agents during task execution. Tool usage should be handled internally within the Node Service, not by inserting additional nodes into the workflow graph for each tool call. Future enhancements and agent implementations must follow this separation of concerns.

## Completed
* [x] Refactored server code:
  * Extracted reusable server logic into `package node`.
  * Created a dedicated entry point at `services/node/cmd/server.go` with `package main`.
  * Resolved package conflicts and improved project modularity.

## Next Steps
* [ ] Pass the available tools to the OpenAI/Openrouter API when invoking an agent
* [x] Add more unit tests for `package node` components.
* [ ] Document the new project structure in `README.md`.
* [ ] Document and test agent tool integration: Ensure agents can invoke tools as part of their execution, and that this is handled within the Node Service harness.

* [ ] **(High Priority)** Implement MCP tool invocation in agent harness
    - See DESIGN.md for architectural details and requirements.
    - [ ] Implement mechanism for agent harness to discover and invoke MCP service tools as part of node execution.
    - [ ] Integrate MCP tool results into node/task outputs and workflow edits as needed.
    - [ ] Handle errors, extensibility, and security as described in DESIGN.md.
    - [ ] Add tests and documentation for MCP tool invocation in the Node Service.
---

* [ ] **(High Priority - CONV-003 Part 2)** Develop base agent/node framework/SDK:
    * [x] Implement Protobuf communication layer (server side for receiving tasks).
    * [ ] Standardize task execution lifecycle (receive, process, report).
    * [ ] Include helper functions for generating Protobuf outputs.
* [ ] **(High Priority - CONV-002)** Implement initial agent roles:
    * [ ] `PlanningAgent`: Takes a high-level goal, outputs a workflow graph definition.
    * [ ] `WorkerAgent`: Takes a specific task definition, executes it, returns structured result. Needs mechanism to map task types to capabilities (e.g., code execution, text generation, data analysis).
    * [ ] `SupervisorAgent`: Monitors workflow events, potentially intervenes or adjusts plans.
* [ ] Integrate LLM interaction logic for relevant agents.
* [ ] Develop capability for agents to generate code that produces Protobuf outputs.
* [ ] Implement secure handling for any necessary credentials or API keys.
* [ ] Integrate with [[protos/TODO.md|Protobuf definitions]] once stable.
