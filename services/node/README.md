# Node Executor Service (Monorepo)

**Purpose:** This repository contains the implementations for the various types of "nodes" or agents that execute tasks within the AI Society workflows managed by the [[services/workflow/README.md|Workflow Service]].

**Core Responsibilities:**
*   Implement specific agent roles (e.g., Planning, Worker, Supervisor, Critic, Toolsmith).
*   Receive task assignments from the Workflow Service via [[protos/README.md|Protocol Buffers]].
*   Execute the assigned task (which may involve invoking LLMs, running code, calling external tools, or delegating sub-tasks).
*   Generate structured output, often by creating Protocol Buffer messages.
*   Report status and results back to the Workflow Service.
*   Potentially manage local state or context relevant to its role/task.

**Structure:** Likely a monorepo containing:
*   A shared library/SDK for interacting with the Workflow Service and handling Protobufs.
*   Separate services/packages for each distinct agent role or type.

**Current Tasks:** See [[services/node/TODO.md]].
