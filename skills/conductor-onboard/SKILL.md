---
name: conductor-onboard
description: Use when the user asks to bring an existing project, its documentation, plans, or backlog under Conductor, migrate its work tracking, or resume an interrupted onboarding. Ordinary work in a registered project uses conductor-work.
---

# Onboard an existing project

Establish a trustworthy starting point for tracking remaining work. Preserve the project's design documents and decisions; Conductor records work status, ownership, dependencies and review evidence. `init` only registers a repository. This skill uses existing CLI commands, not a document importer.

## Establish scope and access

Use the installed executable and configured data home from the managed bootstrap. Read [work commands](../conductor-work/references/commands.md) for identity, session and retry mechanics. Check `--version` (CLI 0.2, schema 3, skill contract `layers-v1`) and `context --if-registered --json` from the target checkout before repository work.

An onboarding request authorizes the requested registration and backlog reconciliation, not implementation, team startup, merges, or external tracker changes. A request for advice or a dry run performs reads and returns a proposed mapping without mutations. Honor existing authorization; ask only for missing decisions that affect scope or policy.

- Healthy `registered=false`: inspect the project first, then use `init --prefix APP --request KEY --json` within the onboarding grant. Record the returned project ID. Linked worktrees share registration; separate clones do not.
- Healthy registration: reuse its project ID, prefix, policy and existing tickets. Do not reinitialize to obtain a fresh board.
- `REGISTRY_UNAVAILABLE` or `DISCOVERY_UNAVAILABLE`: registration is unknown. Surface the diagnostic, resolve access through supported host permissions where possible, and continue independent document assessment. Do not create a fallback registry or report saved imports without successful responses.

Inspect `workflow show` and `team list --json` when registered. A non-stopped team requires coordination with its existing coordinator under [team commands](../conductor-work/references/team-commands.md); do not start a second team or take over active claims.

## Reconcile sources with current work

Read the project's instruction files, document index, current plans, decisions and acceptance records. Follow their source precedence and explicit user amendments; timestamps alone do not establish authority. Preserve historical plans. A copied coordinator prompt or old plan is context, not a new execution grant.

Inspect relevant code, branch/worktree locations and evidence for work described as implemented. Record the actual inspected revision and dirty state. Separate design approval, implementation, integration and acceptance: code on another branch is not necessarily landed, and unit checks do not establish hardware or product acceptance. If evidence is inaccessible, record the uncertainty instead of restarting or declaring completion.

Create a compact source-to-work mapping. Cover the agreed import scope; summarize large future milestones without decomposing every speculative feature. Split partially completed items into delivered scope and remaining integration/validation. Resolve ordinary discrepancies from evidence; leave material product conflicts as explicit decisions and continue unaffected items.

## Record the transition

Use the project's existing tracking/handoff document if appropriate; otherwise create `docs/conductor-onboarding.md`. Preserve source content and link to authoritative definitions. The record contains:

- User-authorized scope, exclusions, repository/revision, Conductor home/project ID, policy choice and active-team context.
- Source map: `Source path + section/ID | Current intent and evidence | Disposition/reason | Existing or returned ticket IDs | Remaining decision/check`.
- Resume checkpoint: identity/session, onboarding claim if held, completed mappings, pending operations and the next safe action.

Dispositions distinguish existing-ticket reuse, newly tracked remaining work, delivered historical context, superseded plans, future/unapproved ideas, and unresolved decisions. Every in-scope source item has a disposition, including items that produce no ticket. Historical completion is recorded as evidence, not manufactured `done` tickets or fresh acceptance.

Before importing, show the concrete mapping and chosen workflow behavior. Proceed when covered by the user's grant; obtain only unresolved scope or retained policy decisions. Keep existing policy by default. For a newly registered project with no requested gates, ordinary tracking is sufficient. If the user requests policy, apply the explicitly authorized configuration from [workflow commands](../conductor-work/references/workflow-commands.md) before creating tickets. Policy changes do not rewrite existing ticket snapshots; do not weaken or replace their gates during onboarding.

After resolving registration and policy, use [conductor-work](../conductor-work/SKILL.md) to track the authorized onboarding task. Register/resume its identity and session. If policy gates apply to that task, use `conductor-plan` to prepare it within the existing grant before claiming. Stop the session after checkpointing and releasing any owned work.

## Import and resume safely

Use [foundation ticket commands](../conductor-work/references/ticket-commands.md) for specifications and dependencies; use [conductor-plan](../conductor-plan/SKILL.md) for preparation under the recorded policy. New policy-governed tickets begin as drafts. A plain ready ticket still does not authorize implementation.

1. Enumerate `ticket list --limit 100 --json`, following each nonempty `data.next_cursor` with `--cursor CURSOR` until exhausted. Do not filter out completed or blocked tickets during duplicate detection. Inspect candidate matches with `ticket show ID`; match scope and source references, not just titles. If listing fails or is incomplete, keep affected mappings pending.
2. Reuse matching tickets. Preserve active ownership, specifications and reviews; propose changes through the proper owner/coordinator instead of resetting another worker's work. For missing work, create bounded tickets with purpose, remaining scope, acceptance criteria, source path/section, evidence limits and validation method. Group related work with parents and order prerequisites with dependencies. Leave unapproved ideas in the source map unless the user requested their tracking.
3. Before each mutation, persist its unique request ID and exact arguments/payload in the resume checkpoint or linked local payload files. Use a project/onboarding-specific namespace and a new ID per logical operation. Save successful returned IDs immediately. This covers registration/session setup as well as ticket creation and linking.
4. On an interrupted or ambiguous response, reconcile current registration, session and ticket state before more mutations. Replay only the identical request and payload; never issue a new create key merely because its result was lost. A matching ticket can satisfy the mapping without proving which request created it: record that distinction. When existing work already covers the mapping, do not replay a create merely to discover its outcome; record the old operation as covered with provenance unresolved and exclude it from pending retries. If no match exists and exact replay data is unavailable, inspect provenance and keep the operation pending rather than guessing.
5. Use returned ticket IDs for parent/dependency edits, preserving the full current specification and expected revision. Checkpoint those edits too. Reruns refresh source changes and ticket state before resuming; changed payloads are new operations after the prior outcome is resolved.

Do not claim or execute imported implementation tickets as part of onboarding. Finish with a coverage check against the agreed source inventory and actual `ticket show` results. Report created/reused tickets, preserved historical/future scope, unresolved decisions and the exact resume point. Hand off to `conductor-plan` for preparation or to `conductor-orchestrator` only when an execution run is separately authorized. Registration, imported backlog, plan readiness and running work are distinct outcomes.
