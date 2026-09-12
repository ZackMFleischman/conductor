# Conductor work board

> Example statuses only; implementation has not started.

| Task | Status | Assigned to | What this task delivers | Waiting on |
| --- | --- | --- | --- | --- |
| T0 · Foundation | Done | core-1 | Create one shared local database that agents can access from their separate working copies of a project. Give each agent an identity so we can track who is doing what. | — |
| T1 · Tickets and claims | In progress | core-1 | Let agents create and assign tasks, reserve a task so another agent cannot take it, report progress, and hand finished work to you for review. Make abandoned tasks available again when needed. | — |
| T2 · Problem reporting | Needs your QA | reporting-1 | Let agents record what went wrong and how they corrected it, preserving the original report so we can later identify recurring problems and improve the workflow. | Your feedback |
| T3 · Work skill and setup | In progress | setup-1 | Provide a simple installation that teaches Codex and Claude Code when and how to use Conductor, without adding instructions to every project or overwriting your existing settings. | — |
| T4 · Combined integration | Waiting | Unassigned | Bring the separately built features together and verify they work as one tool, including when two agents act at the same time or an agent stops unexpectedly. | T1, T2, T3 |
| T5 · Fresh-session validation | Waiting | Unassigned | Package the tool for Windows and try it with newly started Codex and Claude Code agents. Confirm they discover Conductor and use it to coordinate real work across separate working copies. | T4 |
