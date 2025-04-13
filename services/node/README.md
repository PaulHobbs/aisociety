# Node Executor Service 
**Purpose:** This repository contains the implementations for the various types of "nodes" or agents that execute tasks within the AI Society workflows managed by the [[services/workflow/README.md|Workflow Service]].

**Agent Harness & Tool Execution:**
The Node Service is responsible for providing the agent harness (the runtime environment for agents) and for executing any tools or external resources required by agents during task execution. This design ensures a clear separation of concerns: the Workflow Service orchestrates *what* should run and *when*, while the Node Service handles *how* the assigned work is performed, including all tool invocations. This approach improves efficiency, keeps the workflow graph focused on high-level logic, and allows agents to use tools as internal implementation details without unnecessary orchestration overhead.
**Purpose:** This repository contains the implementations for the various types of "nodes" or agents that execute tasks within the AI Society workflows managed by the [[services/workflow/README.md|Workflow Service]].

**Core Responsibilities:**
*   Implement specific agent roles (e.g., Planning, Worker, Supervisor, Critic, Toolsmith).
*   Receive task assignments from the Workflow Service via [[protos/README.md|Protocol Buffers]].
*   Execute the assigned task (which may involve invoking LLMs, running code, calling external tools, or delegating sub-tasks).
    - The Node Service provides the runtime harness for agents and manages all tool execution required by the agent as part of its task. Tool usage is handled internally within the Node Service, not by inserting additional nodes into the workflow graph for each tool call.
*   Generate structured output, often by creating Protocol Buffer messages.
*   Report status and results back to the Workflow Service.
*   Potentially manage local state or context relevant to its role/task.

**Structure:** Likely a monorepo containing:
*   A shared library/SDK for interacting with the Workflow Service and handling Protobufs.
*   Separate services/packages for each distinct agent role or type.

**Current Tasks:** See [[services/node/TODO.md]].

---

## MCP Tool Discovery & Registry API

The Node Service exposes an HTTP endpoint for tool discovery:

- `GET /mcp/tools`: Returns a JSON array of all registered MCP tools, including their name, version, description, and input/output schemas.

Tools can be registered dynamically at runtime using the `MCPToolRegistry` interface (see `mcp_tools.go`). The default implementation is `InMemoryToolRegistry`, but custom registries can be implemented for persistent or distributed scenarios.

Schema validation is enforced on registration: only supported types (`string`, `int`, `bool`, `float`) are allowed in input/output schemas. See `TestInMemoryToolRegistry_SchemaValidation` for validation edge cases.


## MCP Tool Invocation

### Overview

The Node Service supports invoking Model Context Protocol (MCP) tools as part of agent task execution. MCP tools are modular, discoverable, and can be registered dynamically. Each tool defines its own input and output schema, and can be invoked by agents to perform specialized actions (e.g., summarization, data transformation, external API calls).

### How MCP Tool Invocation Works

- **Tool Registry:** Tools are registered in an `MCPToolRegistry` (see `mcp_tools.go`). The registry provides discovery and lookup of available tools.
- **Tool Adapter:** The `MCPToolAdapter` interface defines how a tool is invoked. The default/mock implementation simply echoes input for testing, but real adapters can call external services or run custom logic. Adapters can be swapped at runtime using the `ToolAdapterSelector` function.
- **Invocation Flow:** When an agent's assigned task requests a tool (e.g., via a goal like `Call: tool-name`), the Node Service:
  1. Looks up the tool in the registry.
  2. Validates input against the tool's schema.
  3. Invokes the tool via the adapter.
  4. Returns the result in the task's output.

#### Adapter Extensibility & Error Handling

Adapters can implement custom logic, error propagation, timeouts, and partial result handling. See the following tests in `mcp_tools_test.go`:
- `TestInvokeMCPTool_AdapterError`: Ensures adapter errors are propagated to the caller.
- `TestInvokeMCPTool_AdapterTimeout`: Ensures context timeouts are handled and surfaced.
- `TestInvokeMCPTool_AdapterPartialResult`: Supports partial/incomplete results (e.g., status "PARTIAL").

### Adding a New MCP Tool

1. **Define the Tool:**
   - Create a new `MCPTool` struct with a unique name, description, and input/output schema.
2. **Register the Tool:**
   - Add the tool to the `InMemoryToolRegistry` (or your custom registry implementation).
   - Example (see `mcp_tools.go`):
     ```go
     tool := MCPTool{
         Name: "my-tool",
         Description: "Does something useful",
         InputSchema: map[string]string{"param1": "string"},
         OutputSchema: map[string]string{"result": "string"},
     }
     registry := NewInMemoryToolRegistry([]MCPTool{tool})
     ```
3. **Implement the Adapter (Optional):**
   - If your tool needs custom logic, implement the `MCPToolAdapter` interface.
   - For testing, use `MockToolAdapter`.
## Tool Surfacing for LLM Agents

When invoking LLM-based agents (e.g., via OpenAI or OpenRouter APIs), available MCP tools are surfaced as function/tool schemas using the OpenAI-compatible format. The conversion is handled by `MCPToolToOpenAIFunctionSchema` (see `agent.go`), which maps each tool's input schema to JSON Schema types.

This enables LLM agents to "see" and invoke available tools at runtime, with full schema validation and type safety. See `TestMCPToolToOpenAIFunctionSchema` in `mcp_tools_test.go` for coverage.


### Running MCP Tool Invocation Tests

Comprehensive tests for MCP tool invocation (success, error, and edge cases) are in `node_test.go` and `mcp_tools_test.go`. These include:
- Registry dynamic registration/versioning
- Schema validation
- HTTP endpoint for tool discovery
- Adapter error, timeout, and partial result handling
- Tool schema surfacing for LLM agents

To run all tests:

```sh
make test-pure
```

This command runs all Node Service unit/integration tests, including MCP tool registry, discovery, adapter, and invocation scenarios.
