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
