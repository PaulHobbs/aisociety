# Protocol Buffers

**Purpose:** This directory contains the Protocol Buffer (`.proto`) definitions used for all strictly typed communication between AI Society agents and services.

**Rationale:** Using Protobufs ensures:
*   **Type Safety:** Reduces errors from ambiguous or malformed messages.
*   **Interoperability:** Allows agents/services potentially written in different languages/frameworks to communicate reliably.
*   **Structure:** Enforces a clear schema for data exchange (tasks, results, status updates, events, configuration).
*   **Efficiency:** Provides a compact binary wire format.

**Core Components:**
*   Task definitions and results structures.
*   Agent/Node configuration and status messages.
*   Workflow state updates and events.
*   Potentially messages for governance proposals, votes, knowledge submission.

**Relation to Architecture:** This is a foundational element of the chosen architecture, enabling communication between the [[services/workflow/README.md|Workflow Service]] and the various [[services/node/README.md|Node Executors]]. See [[services/workflow/Architecture.md]].

**Current Tasks:** See [[protos/TODO.md]].
