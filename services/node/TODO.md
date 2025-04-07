# Node Executor TODO

## Completed
* [x] Refactored server code:
  * Extracted reusable server logic into `package node`.
  * Created a dedicated entry point at `services/node/cmd/server.go` with `package main`.
  * Resolved package conflicts and improved project modularity.

## Next Steps
* [ ] Add more unit tests for `package node` components.
* [ ] Document the new project structure in `README.md`.

---

* [ ] **(High Priority - CONV-003 Part 2)** Develop base agent/node framework/SDK:
    * [ ] Implement Protobuf communication layer (server side for receiving tasks).
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
