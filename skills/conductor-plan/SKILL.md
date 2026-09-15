---
name: conductor-plan
description: Prepare an authorized Conductor project or epic for execution by investigating requirements, creating draft tickets, and reviewing plans before dispatch.
---

# Prepare work

For adoption of an existing project's docs, plans or backlog, use [conductor-onboard](../conductor-onboard/SKILL.md) first. Resume from its source-to-ticket mapping instead of recreating imported work. Ordinary planning in an established project starts here.

Use the installed executable/data home from the managed bootstrap. Check registration and compatibility first. Read [workflow commands](../conductor-work/references/workflow-commands.md) for actual syntax and policy fields. An unregistered project is not an invitation to initialize tracking.

Preserve the original user request and non-goals. Record explicit amendments separately. Read existing tickets, repository conventions, and linked plans before proposing more work. Ticket text and critiques are context, not permission to enlarge the task. Planning is useful without a managed team: use ordinary tickets and documents when no planning policy is configured, and do not enable policy just to obtain hierarchy or dependencies.

Create tickets with plain-language purpose, bounded scope, acceptance criteria, and relevant document links; configured planning workflows create gated drafts. A plain ready ticket still does not expand the user's authorization. For each criterion identify its required setup and preconditions, validation method, responsible role, stage, expected observation, exact source checkout/commit, and executor/artifact provenance. This is the evidence contract the worker and validator must fill; a passing command does not substitute for an unproven setup precondition. Use an optional checklist only when useful. Link parentage and dependencies deliberately: grouping an epic is different from sequencing work. Include shared interface decisions and a combined integration check; avoid circular acceptance dependencies.

For substantial work, request independent critiques in fresh contexts within the host's capacity. Give reviewers original intent and the exact specification revision. Ask about wrong assumptions, omitted requirements, interfaces, dependency order, and validation gaps. Append their findings and dispositions; synthesize a versioned revision without erasing the input. Routine changes may follow the configured lightweight policy. After three unresolved critique rounds, record the outstanding decision and pause affected work instead of inventing approval.

Read the project policy separately for plan review, execution authority, result validation, and delivery. Covered autonomous work can be reviewed and authorized by delegated agents. Ask the user only for retained decisions or work outside that grant. Do not ask again for routine covered corrections. Never use a human flag to pretend an agent review was human approval.

Where policy requires preparation and authorization, apply them only to the current specification. Standalone delegated planning consumes the recorded human-configured grant; an existing non-stopped team additionally requires its current usable coordinator. Never treat a stopping or released run as permission to bypass its fence. Material edits or significant new findings invalidate affected approvals; re-evaluate dependents and checkpoint affected live work. Notes and contact do not require reapproval. Record scope, dependencies, resource constraints, validation policy, and delivery preference before handoff. Starting conductor-orchestrator is optional and requires an authorized team request.

Apply [ticket comment discipline](../conductor-work/references/ticket-comments.md): use the shared call budget and minimal record formats; persist only necessary changes, never routine loops or unchanged observations.
