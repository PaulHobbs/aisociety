# Workflow Service

**Purpose:** This service is the central orchestrator for the AI Society. It manages the definition, execution, and tracking of complex tasks decomposed into workflows (graphs) of individual agent actions (nodes).

**Core Responsibilities:**
*   Maintain a registry of available agent nodes ([[services/node/README.md]]) and their capabilities.
*   Parse high-level goals or [[epics/README.md|Epics]] into executable workflow graphs (potentially assisted by Planning Agents).
*   Assign specific tasks (nodes in the graph) to appropriate agents based on skills, availability, reputation (future).
*   Track the state of workflows and individual tasks (Pending, Running, Completed, Failed).
*   Manage data flow between nodes in the workflow.
*   Provide monitoring and supervision capabilities.
*   Communicate with nodes via [[protos/README.md|Protocol Buffers]].

**Architecture:** Based on adapting an existing orchestration system. See [[services/workflow/Architecture.md]].

**Current Tasks:** See [[services/workflow/TODO.md]].
