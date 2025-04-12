# Example Workflow: Document Summarization with Tool Usage

This example demonstrates a workflow where a node (agent) uses an external tool as part of its execution. The tool invocation is handled internally by the Node Service, not as a separate node in the workflow graph.

## Workflow Structure

- **Goal:** Summarize a provided document and review the summary.
- **Workflow Graph:** Two nodes in sequence.

## Nodes

1. **Summarize Document Node**
   - **Agent:** Summarizer
   - **Task:** Generate a concise summary of the input document.
   - **Tool Usage:** Calls an external summarization tool (e.g., a Python script or API) as part of its execution.
   - **Output:** Document summary.

2. **Review Summary Node**
   - **Agent:** Reviewer
   - **Task:** Review the generated summary for accuracy and clarity.
   - **Input:** Output from Summarize Document Node.
   - **Output:** Reviewed summary with comments.

## Example DAG

```
[Summarize Document Node] --> [Review Summary Node]
```

## Notes

- The Summarizer agent invokes the summarization tool internally within the Node Service.
- The Workflow Service is unaware of the specific tool usage; it only tracks the high-level workflow steps and their dependencies.
- This approach keeps the workflow graph focused on business logic, while the Node Service manages all execution details.