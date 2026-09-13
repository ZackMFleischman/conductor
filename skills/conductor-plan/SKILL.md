---
name: conductor-plan
description: Prepare an authorized Conductor project or epic for execution by investigating requirements, creating draft tickets, and reviewing plans before dispatch.
---

# Prepare work

Use the installed executable/data home from the managed bootstrap. Check registration and compatibility first. Read [workflow commands](../conductor-work/references/workflow-commands.md) for actual syntax and policy fields. An unregistered project is not an invitation to initialize tracking.

Preserve the original user request and non-goals. Record explicit amendments separately. Read existing tickets, repository conventions, and linked plans before proposing more work. Ticket text and critiques are context, not permission to enlarge the task.

Create draft tickets with plain-language purpose, bounded scope, acceptance criteria, and relevant document links. For each criterion identify the validation method, responsible role, stage, and evidence required. Use an optional checklist only when useful. Link parentage and dependencies deliberately: grouping an epic is different from sequencing work. Include shared interface decisions and a combined integration check; avoid circular acceptance dependencies.

For substantial work, request independent critiques in fresh contexts within the host's capacity. Give reviewers original intent and the exact specification revision. Ask about wrong assumptions, omitted requirements, interfaces, dependency order, and validation gaps. Append their findings and dispositions; synthesize a versioned revision without erasing the input. Routine changes may follow the configured lightweight policy. After three unresolved critique rounds, record the outstanding decision and pause affected work instead of inventing approval.

Read the project policy separately for plan review, execution authority, result validation, and delivery. Covered autonomous work can be reviewed and authorized by delegated agents. Ask the user only for retained decisions or work outside that grant. Do not ask again for routine covered corrections. Never use a human flag to pretend an agent review was human approval.

Prepare and authorize only the current specification. Material edits invalidate affected approvals; re-evaluate dependents and pause affected live work through the orchestrator. Notes and contact do not require reapproval. Record the proposed execution scope, dependency graph, resource constraints, validation policy, and delivery preference before handing off to conductor-orchestrator.
