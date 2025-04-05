# Workflow Service Architecture

**Foundation:** This service adapts the core concepts and potentially codebase of the existing orchestration system.

**Key Architectural Decisions:**
*   **Workflow Representation:** Workflows are defined as Directed Acyclic Graphs (DAGs) where nodes represent tasks and edges represent dependencies and data flow.
*   **Node Abstraction:** Agent instances ([[services/node/README.md]]) are treated as abstract execution nodes/services. The Workflow service interacts with them via a standardized API.
*   **Communication Protocol:** All interactions between the Workflow Service and Node Executors use [[protos/README.md|Protocol Buffers]] over a suitable transport (e.g., gRPC, message queue).
*   **State Management:** The service maintains persistent state for all active and historical workflows, including task status, assignments, inputs, and outputs.
*   **Event-Driven:** Key events (task assignment, completion, failure) are generated and can be used for monitoring, supervision, and potentially triggering subsequent actions or workflows.
*   **Extensibility:** Designed to support various types of nodes, including AI agents, traditional software tools, and human-in-the-loop interfaces.

**Core Components (Conceptual):**
*   Workflow Definition Parser/Builder
*   Scheduler/Task Assigner
*   State Tracker Database
*   Node Communication Interface (Protobuf-based)
*   Event Bus Emitter
*   Monitoring/Supervision API
