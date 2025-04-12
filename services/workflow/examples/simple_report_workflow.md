# Example Workflow: Simple Report Generation

This example demonstrates a basic multi-step workflow managed by the Workflow Service, with each step executed by a different agent via the Node Service.

## Workflow Structure

- **Goal:** Generate a research report on a given topic.
- **Workflow Graph:** Linear sequence of nodes.

## Nodes

1. **Research Node**
   - **Agent:** Researcher
   - **Task:** Gather information and key facts about the topic.
   - **Output:** Research notes.

2. **Draft Node**
   - **Agent:** Writer
   - **Task:** Write a draft report using the research notes.
   - **Input:** Output from Research Node.
   - **Output:** Draft report.

3. **Review Node**
   - **Agent:** Reviewer
   - **Task:** Review and provide feedback on the draft report.
   - **Input:** Output from Draft Node.
   - **Output:** Reviewed report with comments.

## Example DAG

```
[Research Node] --> [Draft Node] --> [Review Node]
```

## Notes

- Each node is assigned to a specific agent role.
- The Node Service executes each node, handling all agent logic and any required tool usage internally.
- The Workflow Service manages the orchestration, dependencies, and state transitions.