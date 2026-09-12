# Conductor work board

| Task | Status | Assigned to | What this task delivers | Waiting on |
| --- | --- | --- | --- | --- |
| T0 · Foundation | Done | foundation-1 | Create one shared local database agents can access from separate working copies. Give each agent an identity so we can track who is doing what. | — |
| T1 · Tickets and claims | Done | tickets-1 | Let agents create and assign tasks, reserve work, report progress, and hand it to you for review. Recover abandoned tasks safely. | — |
| T2 · Problem reporting | Done | reporting-1 | Record what went wrong and how it was corrected, preserving the original report for later workflow improvements. | — |
| T3 · Work skill and setup | Done | setup-1 | Teach Codex and Claude Code when and how to use Conductor through a simple installation that preserves existing settings. | Fresh-session acceptance in T5 |
| T4 · CON-1 · Combined integration | Ready for QA | coordinator-1 | Combine the separately built features and verify they work together, including when agents compete for work or stop unexpectedly. | Human review |
| T5 · CON-2 · Fresh-session validation | Ready for QA | host-validator-1 | Package the Windows tool and verify newly started Codex and Claude agents discover it and coordinate real work across separate working copies. | Human review; Linux runtime is a later increment |
| T6 · CON-3 · Team workflow design | Design reviewed | coordinator-1 | Define how agents plan and review work before coding, start fresh workers for independent tasks, and turn workflow problems into validated improvements. | Core/skill implementation; case study awaits approval |
| T7 · CON-4 · Workflow integration | In progress | coordinator-1 | Combine the new workflow features, install their skills, and independently verify that agents can use them before the sample project begins. | T8, T9, T10 |
| T8 · CON-5 · Prepared tickets and validation | In progress | workflow-tickets-1 | Let agents plan and review tickets before coding, respect dependencies, and use automated, independent-agent, or human validation as configured. | — |
| T9 · CON-6 · Managed agent team | In progress | workflow-team-1 | Track which agents were launched, confirm they are ready, recover interrupted coordination, and limit concurrency to available capacity. | — |
| T10 · CON-7 · Workflow improvement | In progress | workflow-retro-1 | Turn problem reports into deduplicated improvement tickets, defer work to milestones, and record whether fixes helped. | T8 for linked draft tickets |
