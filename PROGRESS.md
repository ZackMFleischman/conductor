# Conductor work board

| Task | Status | Assigned to | What this task delivers | Waiting on |
| --- | --- | --- | --- | --- |
| T0 · Foundation | Done | foundation-1 | Create one shared local database agents can access from separate working copies. Give each agent an identity so we can track who is doing what. | — |
| T1 · Tickets and claims | Code review | tickets-1 / reviewer | Let agents create and assign tasks, reserve work, report progress, and hand it to you for review. Recover abandoned tasks safely. | Independent review |
| T2 · Problem reporting | In progress | reporting-1 | Record what went wrong and how it was corrected, preserving the original report for later workflow improvements. | — |
| T3 · Work skill and setup | Done | setup-1 | Teach Codex and Claude Code when and how to use Conductor through a simple installation that preserves existing settings. | Real-session acceptance in T5 |
| T4 · Combined integration | In progress | integration-1 / coordinator | Combine the separately built features and verify they work together, including when agents compete for work or stop unexpectedly. | T1 and T2 reviews before final checks |
| T5 · Fresh-session validation | In progress | packaging-1 / coordinator | Package the Windows tool and verify newly started agents discover it and coordinate real work across separate working copies. | T4 for live testing; Claude sign-in |
