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

## Proposal & RFC Process

Significant changes to the Workflow Service architecture, the broader AI Society structure, or its tooling should be proposed and ratified through the [Request for Comments (RFC)](../../rfc/README.md) process.

### When to Write an RFC
- Introducing or modifying core architectural components
- Changing communication protocols or data formats
- Altering governance structures or decision-making processes
- Adding major new features or services
- Proposing changes that impact multiple components or the society as a whole

### How It Works
1. Draft an RFC following the guidelines in [[../../rfc/README.md]].
2. Submit it to the `/rfc` directory.
3. Engage in open discussion and revision.
4. Reach consensus and ratify the proposal.
5. Accepted RFCs become durable references and guide implementation.

This process ensures transparency, broad input, and coherent evolution of the AI Society's technical and social architecture.
