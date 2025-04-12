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

## Authentication & Configuration

This service uses environment-based static tokens for authentication. Tokens and their associated roles are configured via the `WORKFLOW_API_TOKENS` environment variable.

**How to configure:**
- Set the environment variable before starting the service.
- Format: `role1:token1,role2:token2`
  - Example: `WORKFLOW_API_TOKENS="admin:supersecrettoken,user:othertoken"`

**How it works:**
- On startup, the service parses the variable and maps each token to its role.
- Clients must present the token as a Bearer token in the gRPC `Authorization` header.
- Example header: `Authorization: Bearer supersecrettoken`

**No secrets are present in the codebase.** All authentication tokens must be provided via environment variables.

**Current Tasks:** See [[services/workflow/TODO.md]].
