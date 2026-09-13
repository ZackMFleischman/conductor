# Conductor work board

| Task | Status | Assigned to | What this task delivers | Waiting on |
| --- | --- | --- | --- | --- |
| T0 · Foundation | Done | foundation-1 | Create one shared local database agents can access from separate working copies. Give each agent an identity so we can track who is doing what. | — |
| T1 · Tickets and claims | Done | tickets-1 | Let agents create and assign tasks, reserve work, report progress, and hand it to you for review. Recover abandoned tasks safely. | — |
| T2 · Problem reporting | Done | reporting-1 | Record what went wrong and how it was corrected, preserving the original report for later workflow improvements. | — |
| T3 · Work skill and setup | Installed; validated | setup-1 | Teach Codex and Claude Code when and how to use Conductor through six optional skills and an installation that preserves existing settings. | — |
| T4 · CON-1 · Combined integration | Ready for QA | coordinator-1 | Combine the separately built features and verify they work together, including when agents compete for work or stop unexpectedly. | Human review |
| T5 · CON-2 · Fresh-session validation | Ready for QA | host-validator-1 | Package the Windows tool and verify newly started Codex and Claude agents discover it and coordinate real work across separate working copies. | Human review; Linux runtime is a later increment |
| T6 · CON-3 · Team workflow design | Reviewed; implemented | coordinator-1 | Define how agents plan and review work before coding, start fresh workers for independent tasks, and turn workflow problems into validated improvements. | Case-study project and scope approval |
| T7 · CON-4 · Workflow integration | Validated; review | coordinator-1 | Combine the workflow features, install their skills, and verify real workers, independent review, and workflow improvement before the sample project begins. | Existing ticket review; case-study project/scope approval |
| T8 · CON-5 · Prepared tickets and validation | Integrated; tested | workflow-tickets-1 | Let agents plan and review tickets before coding, respect dependencies, and use automated, independent-agent, or human validation as configured. | Existing ticket review |
| T9 · CON-6 · Managed agent team | Integrated; tested | workflow-team-1 | Track which agents were launched, confirm they are ready, recover interrupted coordination, and limit concurrency to available capacity. | Existing ticket review |
| T10 · CON-7 · Workflow improvement | Integrated; tested | workflow-retro-1 | Turn problem reports into deduplicated improvement tickets, defer work to milestones, and record whether fixes helped. | Existing ticket review |
| T11 · Layer separation | Installed; tested | layer-foundation-1 / layer-team-1 / layer-policy-1 | Let agents use ordinary tickets, dependencies and result review without requiring our planning workflow or orchestrator. Preserve existing data and optional gates. | — |
| T12 · CON-8 · Portal | Live integration validated | portal-coordinator-1 / portal-live-adapter-1 | Show real tickets on the React and MUI board with assignment, search, parent filtering, project selection and automatic updates. | — |
| T13 · CON-13 · Live portal API | Landed; in review | backend-api-1 | Serve real database snapshots and update the portal when agents change tickets, including changes made while disconnected. Combined browser checks passed. | Legacy ticket acceptance |
| T14 · CON-19 · Parent hierarchy contract | Done | backend-api-1 | Define complete parent ancestry and consistent live snapshots so the portal can filter whole ticket subtrees correctly. | — |
| T15 · CON-21 · Database snapshots | Done; independently accepted | backend-read-model-1 | Read tickets, assignments, dependencies and parents together so the board never mixes different database states. | — |
| T16 · CON-22 · Live portal connection | Done; independently accepted | portal-live-adapter-1 | Connect the board to real projects, refresh when tickets change, and recover safely from connection failures. | — |
| T17 · CON-25 · Compact descriptions | Done; independently accepted | backend-api-1 | Keep long descriptions to four lines and let the viewer expand each card to read the full text. | — |
| T18 · CON-26 · Active claim visibility | Ready | Unassigned | Show whether a ticket has an active claim separately from its assigned agent. | Future API and frontend increment |
