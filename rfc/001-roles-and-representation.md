# RFC 001: Roles in the AI Society and Their Technical Representation

## Status
Draft

## Authors
AI Society Core Team

## Overview
This RFC proposes a structured framework for defining **roles** within the AI Society, along with a technical schema for representing these roles in agent identities and communication protocols.

## Motivation
To enable a persistent, autonomous, and evolving AI society, agents must assume **specialized roles** that support complex collaboration, governance, knowledge management, and self-improvement. Explicitly defining these roles and representing them in a structured, machine-readable way will:

- Facilitate **clear task delegation** and workflow orchestration.
- Support **governance** by distinguishing decision-making authorities.
- Enable **adaptive role assignment** and evolution over time.
- Improve **transparency** and **auditability** of agent actions.
- Provide a foundation for **protocols** and **services** to reason about agent capabilities and permissions.

## Proposed Roles

The following initial set of roles is proposed. These can be extended or refined via future RFCs.

| Role Name       | Description                                                      | Example Functions                                  |
|-----------------|------------------------------------------------------------------|---------------------------------------------------|
| **Planner**     | Decomposes goals into workflows and assigns tasks                | Task graph design, goal analysis                  |
| **Worker**      | Executes assigned tasks                                          | Code generation, data processing                  |
| **Supervisor**  | Monitors task execution, provides feedback, escalates issues     | Quality control, error handling                   |
| **Curator**     | Manages knowledge base, validates and integrates new knowledge   | Book updates, fact-checking                       |
| **Governor**    | Participates in governance, voting, and policy enforcement       | Approving RFCs, enforcing constitution            |
| **Reflector**   | Analyzes society performance, suggests improvements              | Metrics analysis, proposing governance changes    |
| **Communicator**| Interfaces with external systems or humans                       | API calls, user interaction                       |
| **Maintainer**  | Manages infrastructure, services, and agent lifecycle            | Service deployment, health monitoring             |

*Note:* Agents may hold **multiple roles** simultaneously or switch roles dynamically.

## Technical Representation

### Extending `AgentIdentity`

Building on the existing `AgentIdentity` proto (see `protos/workflow_node.proto`), roles should be represented explicitly and flexibly.

### Role Enum

Define a `Role` enum capturing the core roles:

```proto
enum Role {
  ROLE_UNSPECIFIED = 0;
  PLANNER = 1;
  WORKER = 2;
  SUPERVISOR = 3;
  CURATOR = 4;
  GOVERNOR = 5;
  REFLECTOR = 6;
  COMMUNICATOR = 7;
  MAINTAINER = 8;
}
```

### Updated `AgentIdentity`

Allow agents to have **multiple roles**:

```proto
message AgentIdentity {
  string agent_id = 1;
  repeated Role roles = 2;  // Multiple roles per agent
  string model_type = 3;    // e.g., GPT-4, Claude-3
  map<string, string> metadata = 4; // Optional extra info
}
```

### Role Metadata (Optional)

For richer semantics, roles can be extended with metadata:

- **Capabilities:** What the agent can do.
- **Permissions:** What the agent is allowed to do.
- **Reputation/Trust Scores:** For governance weighting.
- **Lifecycle State:** Active, suspended, retired.

This can be encoded in the `metadata` map or via additional message types in future RFCs.

## Impact

- **Workflow Systems:** Can assign tasks based on roles.
- **Governance:** Enables role-based voting, permissions, and accountability.
- **Knowledge Management:** Curators and Reflectors can be explicitly identified.
- **Security:** Supports access control and audit trails.
- **Adaptability:** Roles can evolve without breaking compatibility.

## Future Work

- Define **role assignment protocols** (manual, automated, reputation-based).
- Specify **role transition mechanisms** (promotion, demotion, delegation).
- Develop **governance models** leveraging roles.
- Extend protobuf schemas with richer role metadata.
- Implement **role-aware services** and workflows.

## Conclusion

Explicitly defining and technically representing roles is foundational for a transparent, adaptable, and persistent AI Society. This RFC provides an initial framework to be refined through implementation and further proposals.