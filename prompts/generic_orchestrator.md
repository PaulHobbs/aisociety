Given 

@/vision.md , @/services/workflow/TODO.md , @/services/node/TODO.md , @/services/scheduler/internal/TODO.md 

Find the most important work area to do next, then use the file read to to read the corresponding docs (DESIGN.md / README.md / implementation.md) before delegating either pre-existing TODO tasks or a new breakdown of tasks into subtasks. Use the exisiting TODO docs to keep track with reports of the project's progress.

When starting on a pre-existing TODO, always delegate to a subtask to first check if these TODOs have been already been done, and have the subtask simply update the TODO if so.

For validation:
- Always ask a subtask for a `make $test` test which can you can use validate their results.
- Don't trust a subtask to validate their work; you need to double check whether follow-up or clean-up work is needed.
- If any tests are failing, you should always leave an uncompleted TODO at the top of the corresponding service's TODO.md.