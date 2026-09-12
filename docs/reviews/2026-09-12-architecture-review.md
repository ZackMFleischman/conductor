# Conductor architecture review

2026-09-12 · Design review only; no software was implemented or tested.

Reviewed these exact current files; the short names below refer to their numbered lines:

- **Main design:** [2026-09-12-conductor-design.md](C:/Users/zFlei/repos/conductor/docs/superpowers/specs/2026-09-12-conductor-design.md).
- **Agent deliverables:** [2026-09-12-conductor-agent-deliverables.md](C:/Users/zFlei/repos/conductor/docs/superpowers/specs/2026-09-12-conductor-agent-deliverables.md).
- **UI design:** [2026-09-12-conductor-ui-design.md](C:/Users/zFlei/repos/conductor/docs/superpowers/specs/2026-09-12-conductor-ui-design.md).

The standard is a lightweight tracker that real agents can install and use across worktrees, with transactional ownership and a small CLI tracer before the React/MUI board.

**Recommendation: conditional go.** The architecture is suitable, but resolve findings 1–4 in the tracer contract before treating it as ready for daily use. These require small decisions and acceptance checks, not a new service, hooks, or the complete target schema. Finding 5 is delivery clarification.

## Findings, ranked

### 1. P1 — An abandoned claim can strand the first tracer's only ticket

**References:** main design lines 96, 120–124, 162, 287–288.

The tracer includes exclusive claims and explicit sessions, but explicit release/takeover/recovery is deferred to slice 2. Claims do not expire. If an agent claims a ticket and loses its conversation/session context, the ticket remains `in_progress` and cannot re-enter the ready queue. Restarting the CLI does not solve that. The proposed first acceptance scenario exercises only the successful handoff path.

**Smallest fix before tracer:** include a human-invoked `release` operation with a reason. Atomically invalidate the claim generation, move the ticket to `ready`, increment revision, and append the event. Require owner mutations to reject the old credential. Keep takeover assignment, quiet-session UI, and richer recovery in slice 2. State explicitly that release does not stop the old process or protect its filesystem writes.

**Acceptance:** abandon a winning session, release its claim, let another session claim, and verify the previous owner cannot submit. No time-based lease or hook is needed.

### 2. P1 — Shared-store access is an unresolved installation dependency

**References:** main design lines 21–22, 46, 80, 168; agent deliverables lines 17, 19, 55–59, 63.

The default store is outside worktrees, while onboarding delegates access to unspecified host-supported settings. In the actual reviewing Codex task, writable roots cover the repository, visualization directory, and temporary storage; `%LOCALAPPDATA%\\Conductor` is outside them. This does not establish that Codex cannot support the proposed layout. It establishes that installing a binary, instructions, and skills alone is insufficient evidence that the advertised tracer will work in fresh sandboxed sessions.

The read-only discovery check also needs an explicit failure policy. If registration lives in an inaccessible store, the CLI cannot establish whether this particular repository is registered. It must not invent either a successful unregistered result or a registered-project result.

**Smallest fix before tracer:** make shared-store access a first-tracer spike and acceptance gate for the actual supported Codex and Claude Code configurations. Record the exact narrow directory-access configuration, its scope across new worktrees/sessions, and any fresh-session requirement. Run `doctor` and a real write from inside each agent's sandbox, including SQLite sidecars and session credential files. Distinguish “registry unavailable; registration unknown” from “readable registry; repository unregistered.” Do not silently initialize a fallback database.

If a host configuration cannot support this, document that limit and choose the smallest workable access approach based on the result. A daemon or hooks are not justified in advance.

### 3. P1 — Claim acquisition and credential caching have an undefined crash boundary

**References:** main design lines 112, 114–116, 135, 162, 201, 272, 287–288.

The claim commits transactionally in SQLite, its token hash is stored there, and the usable credential is cached separately under the session in the data directory. The ordering and recovery contract are unspecified. If the claim commits and the process exits before the cache is published, the ticket has an owner but that owner may not possess the credential required to operate it. A lost successful response creates a similar retry problem. Full idempotent retry coverage is deferred to slice 2, while the target contract promises original results on retry.

**Smallest fix before tracer:** define a recoverable acquisition protocol and include idempotency for claim and submit in the tracer. For example, durably retain a pending session/request credential before attempting the transaction, bind the request to that session and ticket, and reconcile the same request after interruption. An equivalent transactional recovery design is fine. Define how an existing active claim is resumed without manufacturing a new session or guessing ownership from the directory. Limit the early mechanism to the commands whose ambiguous success strands ownership or handoff; broader retry coverage can remain later.

**Acceptance:** interrupt acquisition after the database commit but before normal response/cache completion. The same session must recover its original claim, and a different session must still fail to claim it. Retry a successful submit after losing its response and verify one handoff/event result.

### 4. P1 — The tracer's minimal state transitions are not yet executable as a contract

**References:** main design lines 92–102, 114, 149–158, 287; UI design line 23.

The queue accepts only `ready` tickets, but the tracer does not specify the creation default or an operation to promote a new ticket from `backlog` to `ready`. `ticket ready` is presented as a query. The tracer names submit/accept handoff, while the target human rejection path returns work to ready; it does not explicitly put that rejection operation into the first slice. Two reasonable implementations could therefore create an unclaimable ticket or make a failed QA submission impossible to return for rework without an unrestricted state edit.

**Smallest fix before tracer:** state the creation default and enumerate the few allowed transitions with caller, required revision/claim, and claim-release behavior. A sufficient subset is create-ready; ready → in-progress by claim; in-progress → review by owner submission; review → done by human acceptance; review → ready by human rejection with reason; and the explicit release from finding 1. Keep QA enabled for this tracer if that avoids implementing the no-QA branch immediately. Agent ordinary completion must not act as human acceptance, while acknowledging the existing same-OS-account limitation.

**Acceptance:** create, claim, submit, reject, reclaim, resubmit, and accept using supported CLI commands. Stale acceptance and stale owner submission must fail without changing state or history. No configurable workflow engine is needed.

### 5. P2 — “First release” and the mandatory second slice blur the intended small increments

**References:** main design lines 7, 15, 287–291; agent deliverables lines 46, 63–65; UI design lines 7, 15–23.

The authoritative sequence correctly defines a CLI-only tracer, but line 15 defines first-release success using dependent tickets and browser QA. Slice 2 then bundles dependencies, parent/types, optional checklists, richer context, recovery, broad retries, backup/export, retrospective processing, and Linux validation before the board. This can turn “use the tracer, then add capabilities” into a large second gate despite the UI document's small-board direction.

**Smallest clarification:** name the first installed CLI milestone separately from the later browser milestone. After the tracer, prioritize dependencies/basic parent context and observed usability failures. Allow optional checklist commands and richer retrospective bookkeeping to ship independently of the board. Keep basic report export and an ordinary Markdown retrospective inexpensive; no custom review engine is necessary. This is a sequencing recommendation, not a demand to bring deferred features into the tracer.

## What is sound

- One local SQLite database outside Git checkouts, shared application operations, and a CLI independent of the web process fit the user's one-host workflow. Canonical Git common-directory identity avoids branch and remote-URL identity mistakes.
- Atomic eligibility/claim transactions, revision checks, append-only events, separate assignment and ownership, and explicit limits on filesystem protection address the real coordination problem.
- Dependencies remain separate from parents; cancelled prerequisites remain unsatisfied; optional checklists do not become completion gates. Human QA has a clear intended handoff.
- Bootstrap discovery plus user-level skills is a reasonable proposal that explicitly avoids guarantees of model obedience. No hooks, MCP service, launcher, or repository instruction edits are required by the design.
- Factual problem reporting preserves original evidence and permits corrections. The separate retrospective distinguishes observation from inferred cause and review authorization from fix authorization.
- The React/MUI board precedes Markdown rendering and document resolution. Later worktree-specific live references and snapshots have appropriately explicit provenance and missing-source behavior.

## Tight first acceptance scenario

On Windows, install the tracer and work skill once, register one repository, and start fresh Codex and Claude Code sessions in two linked worktrees with supported sandbox access. Both discover the same project; an unrelated repository returns the defined unregistered result. Create one ready ticket and race two explicit sessions: exactly one wins. Save a progress note and factual problem observation. Exercise interrupted claim recovery, then submit, reject, reclaim, resubmit, and accept. Separately abandon and manually release a claim, verifying the old owner is rejected. Restart the CLI and confirm tickets, ownership history, handoff evidence, and the original problem remain intact.

This is the implementation-readiness gate for the first tracer. Dependencies, retrospective export, Linux portability, the Kanban portal, and document rendering retain their subsequent acceptance milestones. No claim is made here that any unimplemented behavior has passed.
