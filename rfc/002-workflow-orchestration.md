# RFC 002: Scalable Workflow Orchestration in the AI Society

## Status
Draft

## Authors
AI Society Core Team

## Overview
This RFC proposes a framework for using the AI Society's **workflow engine** to accomplish complex, large-scale tasks through **delegation**, **context management**, and **structured orchestration**. It emphasizes that **planning and decomposition** are themselves **dynamic workflow tasks**, enabling recursive, adaptive collaboration.

## Motivation
To tackle ambitious goals, the AI Society must:

- **Decompose** large tasks into manageable subtasks.
- **Delegate** responsibilities to specialized agents.
- **Limit context size** per agent to reduce complexity and token usage.
- **Coordinate** agent interactions effectively.
- **Oversee and evolve** workflows over time.

A structured approach ensures scalability, transparency, and adaptability.

## Principles

- **Modularity:** Break down goals into discrete, well-defined subtasks.
- **Delegation:** Assign subtasks to agents with appropriate roles and expertise.
- **Context Limitation:** Provide each agent only the relevant context needed for its subtask.
- **Structured Communication:** Define clear protocols for information flow between agents and workflow nodes.
- **Transparency:** Maintain an auditable record of task decomposition, delegation, and results.
- **Adaptability:** Allow workflows to evolve based on feedback and changing requirements.
- **Oversight:** Enable monitoring, intervention, and governance throughout execution.
- **Recursive Planning:** Treat planning and decomposition as internal workflow tasks that modify the workflow graph.

## Recursive Workflow Decomposition

- The **workflow engine** represents tasks as a **directed graph** of nodes.
- Each **node** corresponds to a subtask with:
  - A clear **goal**.
  - **Dependencies** on other nodes.
  - An assigned **agent** (with specific roles).
- The **root node** represents the overall goal.
- Nodes can be **recursively decomposed** into child nodes, enabling hierarchical task breakdown.

### Planning as a Workflow Node

- **Automated decomposition** is itself a **planning task**.
- This is represented as a **specialized node** in the workflow graph, assigned to a **Planner** agent.
- The **output** of this planner node is a set of **`NodeEdit`** operations:
  - Adding new child nodes (subtasks).
  - Modifying existing nodes.
  - Reassigning responsibilities.
- This enables **dynamic, in-situ evolution** of the workflow graph during execution.
- Planning can be **recursive**: planner nodes may spawn further planner nodes for subdomains.

## Context Management

- Each agent receives:
  - The **subtask goal**.
  - Relevant **inputs** and **dependencies**.
  - **Role-specific** instructions.
- Agents do **not** require full knowledge of the entire workflow, reducing cognitive and computational load.
- This supports **scalability** and **parallelism**.

## Structured Communication

- Agents communicate via **typed messages** defined in Protocol Buffers:
  - **Task requests and results.**
  - **Status updates.**
  - **Feedback and escalation signals.**
  - **Graph edits** (from planner nodes).
- The workflow engine manages **message routing** based on the graph structure.
- Communication is **logged** for transparency and debugging.

## Oversight and Evolution

- **Supervisory agents** or governance bodies can:
  - Monitor workflow progress.
  - Intervene in case of errors or deadlocks.
  - Approve or reject **NodeEdits** proposed by planner nodes.
  - Adjust task decomposition or delegation dynamically.
- The workflow graph supports **edits** (`NodeEdit`) to evolve workflows over time.
- Historical records enable **reflection** and **continuous improvement**.

## Technical Schema References

- **`workflow_node.proto`** defines:
  - `Node` with `node_id`, `description`, dependencies, assigned `AgentIdentity`, `Task`, `Status`.
  - `AgentIdentity` with roles.
  - `Task` with goal, dependencies, results.
  - `NodeEdit` for graph modifications.
- Future extensions may include:
  - Richer **capability** and **permission** metadata.
  - **Role-based** access and delegation protocols.
  - **Metrics** for workflow performance.

## Future Work

- Develop **planner node** strategies for automated, recursive decomposition.
- Implement **dynamic delegation** based on agent performance and availability.
- Define **escalation protocols** for failed or stalled tasks.
- Integrate **governance** and **reflection** mechanisms.
- Extend schemas for richer context and communication patterns.

## Conclusion

Representing planning and decomposition as **internal workflow nodes** enables the AI Society to dynamically, recursively, and transparently orchestrate complex tasks. This RFC outlines foundational principles and technical directions to realize scalable, adaptive, and evolvable workflows.