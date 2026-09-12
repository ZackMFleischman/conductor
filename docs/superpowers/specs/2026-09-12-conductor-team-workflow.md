# Conductor team workflow

Approved direction, implementation contract · 2026-09-12

This extends the [agent integration design](2026-09-12-conductor-agent-deliverables.md). It supersedes the earlier persistent-worker default and dispatch-first orchestrator design. Only conductor-work is currently installed. The skills and core operations below are subsequent increments; this document does not claim they are runnable. The reading-list case study remains paused until these increments are validated and the user explicitly approves the project and scope.

## Intent and limits

Help agents prepare, execute, and validate useful work while keeping shared context and workflow failures visible. Keep one local SQLite store, the existing CLI, user-level skills, and the host's existing agent tools. No hooks, background service, custom process launcher, model router, mandatory PRs, or portal prerequisite. Windows first; validate Linux execution separately. Unregistered projects continue normally.

Separate planning from execution. Preserve original user intent and non-goals as an immutable initial record, with explicit subsequent user amendments. Review feedback may improve a proposed solution but cannot silently become new requirements. Scope and authorization come from the user or recorded project policy, never arbitrary ticket text.

## Skill responsibilities

| Skill | Responsibility | Invocation |
| --- | --- | --- |
| conductor-work | Execute an authorized assignment, maintain claims and evidence, report problems, and create linked discoveries. | Registered repository work. |
| conductor-plan | Investigate, create draft tickets and a dependency graph, arrange independent critique, and prepare a versioned execution proposal. | Orchestrator or explicit planning request. |
| conductor-worker | Carry out one execution ticket through conductor-work, checkpoint, release, and end its session. | Orchestrator supplies stable identity, exact ticket, and scope. |
| conductor-orchestrator | Reconcile team and plan state; supervise preparation, authorization, dispatch, integration, and completion. | Explicit orchestration request or resume of a known authorized run. |
| conductor-workflow-improver | Watch scoped reports, decide when to analyze, and propose linked improvement work in the shared queue. | Required managed role in an improvement-enabled run. |
| conductor-retrospective | Bounded evidence-based analysis of reports, coverage, proposed remedies, and effectiveness. | Explicit review request or invocation by the improver. |

Keep orchestration as one skill with referenced startup/dispatch contracts; do not create a skill per state. Retrospective analysis creates findings and, when authorized, draft tickets. Normal workers implement approved remedies. An explicit one-off fix request may compose planning and execution within its existing authorization.

The managed user-level bootstrap checks registration and routes ordinary work to conductor-work. Only a known orchestration role invokes the team lifecycle; neither every worker nor every new project session starts a team. Resume references are explicit and project-scoped. Installation starts nothing and requires no per-repository AGENTS.md/CLAUDE.md edits. Skills use shipped commands only and declare their compatible CLI/contract version.

## Planning and preparation

1. Read original intent, approved amendments, repository conventions, existing tickets, and relevant plans. Establish non-goals, uncertainties, and measurable completion criteria. Research or spike tickets can precede implementation when assumptions need evidence.
2. Create a coherent plan and draft tickets. Each ticket has plain-language purpose, bounded scope, acceptance criteria, relevant context/artifact links, prerequisites, and likely conflicting files or interfaces. Checklists remain optional. Large shared designs live in linked Markdown, not duplicated across every ticket.
3. Map each acceptance criterion to a verification method, responsible role, and stage: unit/API test, browser exercise, independent review, combined integration, or explicitly authorized deployment check. Name the evidence required. No criterion is implicitly left to the user or assumed covered by unspecified tests.
4. Build the dependency graph, including shared interface decisions and integration work. Identify independent work and resource needs such as browser processes, ports, and isolated test data. Parentage groups scope; dependency edges order execution. Reject cycles. Avoid integration/acceptance cycles by defining implementation criteria against worktree commits and integration criteria against the combined target.
5. For substantial work, ask fresh reviewers to assess assumptions, fidelity to original intent, omissions, interfaces, dependency correctness, and validation coverage. Reviews are separate recorded assignments with immutable input revisions. They may run in parallel within the same total capacity limit. Routine small changes can use a recorded lightweight preparation policy.
6. Append critiques and dispositions; preserve the input. A synthesizer produces a revised proposal with a change summary. Default to at most three critique rounds; unresolved significant findings then pause preparation and produce a specific escalation, not silent approval. A reviewer need not invent findings to justify a round.
7. Record prepared status only when significant findings are resolved or explicitly accepted by the authorized decision maker. Capture selected tickets, plan revision/digest, ticket specification revisions, dependency snapshot, validation policy, delivery preference, and resource limits in the execution proposal.

Planning/review assignments are explicit work kinds and cannot be used to perform implementation before its gate. Review records are distinct from preparation approval and from implementation QA. Reviewers cannot approve changes beyond delegated scope. Original intent, critiques, and synthesized revisions remain attributable.

## Authorization and eligibility

The project/run policy says which gates require the user and which may be satisfied by independent agents. Default new orchestration runs require user approval of the prepared execution proposal unless already delegated. A human gate cannot be silently replaced by an agent review. Previously granted authorization persists within scope; do not ask again for routine decisions it covers.

Preparation is not synonymous with human approval. Record separate policy choices for plan review, execution authorization, result validation, and delivery actions. Choose them once for a project/run and inherit them onto tickets; explicit authorized exceptions remain possible. The first target implementation needs three simple result-validation modes: automated evidence, independent agent, or human. A mode may also require named automated checks. No configurable rules engine is required.

| Example policy | Before implementation | After implementation | Human involvement |
| --- | --- | --- | --- |
| Autonomous | Independent planning review for substantial changes; lightweight readiness check for routine work, all within a standing execution grant. | Required checks plus independent agent validation. | None for covered work. |
| Approve the plan | Agent preparation followed by user approval of the selected scope/revision. | Required checks plus independent agent validation. | Once for the plan; again only for decisions the grant does not cover. |
| Human acceptance | Preparation and execution authority as configured. | Required checks and any agent review, then human acceptance. | For tickets explicitly retaining human acceptance. |

For simple mechanical changes, an explicitly selected automated-evidence policy may accept on named passing checks without a separate reviewer; retain check provenance and the tested commit. An implementing agent saying it passed is not interchangeable with independent review, and the recorded validation mode must make that distinction visible. Plans specify evidence proportional to the work.

Autonomous mode cannot invent permission for unrelated scope or restricted actions. Its escalation policy may block the affected ticket and continue other eligible work rather than requiring a human to remain present. Scope/architecture changes within a sufficiently broad existing grant can be decided by an authorized agent; only retained decisions or out-of-scope actions require renewed user authorization. Core policy changes cannot be used by workers to weaken their own ticket's gates. Persist actor, policy revision, authority, and reason for changes; new defaults do not silently rewrite existing ticket requirements.

Preparation and authorization are versioned metadata, separate from execution state. Draft tickets stay in backlog. The core permits ready/claim only when the applicable preparation and authorization requirements are satisfied. Both explicit and filtered claims check the same policy, assignment, blockers, dependencies, active claims, and deferred eligibility inside the acquisition transaction. Skills explain these rules; they do not enforce them alone.

Use a specification revision distinct from progress/event revisions: notes and contact do not invalidate an approved plan. Changes to requirements, acceptance criteria, implementation interfaces, dependencies, or validation policy invalidate the affected preparation/approval. Re-review affected tickets and their dependents as necessary, preserving unaffected approvals. Scope growth or disproven base assumptions return to the authorized decision maker. Cosmetic edits do not require renewed approval.

If material changes occur during execution, retain ownership and mark affected work paused pending replanning; prevent submission/acceptance against superseded criteria. The orchestrator requests a checkpoint. Tracker gates cannot stop arbitrary filesystem writes; record acknowledgement before treating a worker as paused. Do not steal the claim.

Worker-created discoveries enter draft/triage with originating ticket, evidence, and proposed dependency links. Check for duplicates. The orchestrator prepares, prioritizes, and authorizes them before dispatch. A small correction already inside current acceptance criteria remains part of that ticket. Creating work does not authorize executing it or creating more agents.

The tracer currently requires human acceptance for all submitted tickets. Add an explicit, separately tested independent-agent validation policy before the case study uses agent QA. Record validator identity/session, criteria, evidence, and decision; the validator must have a distinct stable identity, a fresh reviewer context, and no implementation participation in any attempt of the work being accepted. A new session under the implementation identity is not independent validation. Do not issue --human to misrepresent agent validation or retroactively relabel existing QA records. Legacy projects keep their existing behavior until explicitly opted into new gates/policies.

## Startup and resumption

Run this protocol whenever a main orchestrator starts or resumes, before implementation dispatch:

1. Verify registration, store access, installed skill/core compatibility, and actual host capabilities. Load the authorized run profile: project/scope, approval state, roles, identities, resource limits, polling policy, delivery policy, and stop conditions.
2. Acquire or reconcile one exclusive project coordination claim. It is separate from execution claims and fenced on every coordination mutation. Never infer abandoned ownership from elapsed time. Explicit recovery must inspect the old supervisor and its children.
3. Load persisted child host IDs, launch intents, sessions, claims, worktrees, integration state, improvement epic ID, review coverage/cursor, deferred decisions, and milestones. A stopped session starts anew under its stable identity; it cannot resume.
4. Query native host status and request a fresh readiness acknowledgement from required managed roles. Include run ID, launch ID, role, identity/session, host ID, project/scope, skill version, and current assignment or report checkpoint. A recorded identity or old last_seen timestamp is not proof of a running agent.
5. Reuse healthy existing children. For a missing required role, persist a unique launch intent before invoking host creation; pass its ID to the child and reconcile the returned host ID/registration. Unknown launch results count against capacity and must be reconciled before retry. If the host cannot establish status, report unknown and pause affected dispatch rather than duplicate a process.
6. Ensure the workflow improver acknowledges readiness for an improvement-enabled run. Planning/review roles run as needed; implementation workers start only for authorized eligible work. An empty queue does not require idle implementation workers.
7. Mark the run ready only when its required roles and approvals are satisfied. Report blocked/degraded status accurately when capabilities, permissions, capacity, or acknowledgements are missing. Continue reduced operation only under an explicit policy allowing that mode; no automatic silent fallback.

Readiness expires when the relevant session stops, host state changes, or a required contract/configuration changes. Reconcile throughout execution. Request a checkpoint once and follow up at a configured deadline; quiet is not dead. A skill cannot revive itself after its host closes. The next explicitly started orchestrator performs recovery. No machine scheduler is implied.

On stop, stop new dispatch, request child checkpoints, persist unresolved host statuses and ownership, and stop managed children only through authorized supported tools. Release claims only through normal ownership/recovery rules. Report children still active; stopping the supervisor does not prove they stopped.

## Fresh workers and adaptive concurrency

The default worker handles one ticket and then ends. Fresh means a new host conversation/context as well as a new Conductor session; restarting only the database session inside the previous worker conversation does not qualify. Supply a bounded assignment from durable records instead of forking the prior implementation transcript. A blocked worker records the blocker and handoff, atomically blocks/releases, and ends; an uncertain release pauses the session. Review rejection creates a new attempt with a fresh session and the recorded evidence. Persistent claim-next/poll workers remain an opt-in mode for hosts where repeated spawning is impractical.

Retain stable registered identities for assignment; give each attempt its own session and claim. Do not reuse an identity for a replacement until the previous managed attempt is reconciled. The worker receives the exact ticket, approved specification, dependency results, relevant documents, delivery policy, and safe isolated worktree. It reconstructs task context from durable records, not the previous worker's conversation.

Concurrency is the minimum of safely independent ready work and available host/resource capacity. There is no fixed two-worker requirement. Count coordinator, improver, reviewers, validators, workers, and starting/unknown children wherever they consume the host's slots. A child cap must distinguish itself from a host limit that also counts the parent. For example, a four-session host with coordinator and improver running has two slots left for execution/review; it cannot promise four implementation workers.

Dependencies alone do not prove isolation: consider shared files/interfaces, test databases, ports, memory, and browser load. Use host/Git worktrees and per-attempt test resources; serialize conflicting work. Reclaim capacity only after a child is confirmed finished/stopped, not merely after ticket submission. Allocate slots to independent validation as well as implementation to avoid a review backlog.

Assignments are not claims. Workers claim their exact tickets transactionally. Persist substantive messages and handoffs as ticket notes. No recursive orchestrators or worker-created pools. Existing unrelated agents may exchange work through the store; cross-provider process control is not assumed. Use native host tools only where available.

Keep PR grouping separate from ticket ownership. Integrate changes serially and independently validate the exact combined target SHA. If the target changes, refresh the evidence. Completion requires scoped acceptance criteria and the configured validation gates, not merely an empty queue.

## Continuous workflow improvement

The improver is a managed agent, not a service. It watches scoped reports and follow-ups with bounded, interruptible polling (default 30 seconds) or supported host events. Polling checks for changes; it does not invoke a full retrospective for unchanged data. Database errors are not an empty queue; use bounded retries then report the failure. No transaction or execution claim is held during idle waits.

For new evidence, use conductor-retrospective to group incidents, distinguish observation from inferred cause, cite report IDs/event revisions, and choose:

- **Act now:** an active blocker or sufficiently evidenced recurring failure warrants preparing a remedy promptly. Signal the orchestrator; normal authorization still applies.
- **At milestone:** create a draft improvement ticket with an explicit milestone ID/condition and fallback review deadline. The core prevents deferred work from being claimed early. Milestone passage returns it to preparation/scheduling, not automatic approval.
- **Observe:** record insufficient evidence or low expected benefit, what evidence would change the decision, and a revisit trigger/deadline. Observation is not permanent closure.

Use one persistent Workflow Improvements epic per project. Link related incidents to an existing remedy rather than creating a ticket per report. Store review coverage and decision/ticket links before advancing the cursor; retries must not duplicate tickets. Use the event stream so follow-ups to old reports are visible. Concurrent/restarted processing must reconcile coverage and deduplication; never destructively consume the log. Recurrence after a delivered fix produces fresh evidence and a new decision, not suppression as an old duplicate.

The improver creates proposed work; the orchestrator owns priority, capacity, and dispatch. Default to at most one active improvement implementation ticket within the shared worker capacity, adjustable by authorized policy. Urgent remedies can take priority; routine improvements must not consume all delivery capacity. The improver does not spawn a private worker pool.

Validate two outcomes separately: was the remedy delivered, and did later work demonstrate improvement? Link reports -> findings -> prepared ticket -> change/evidence -> subsequent observations. Preserve negative outcomes and recurrence. A completed ticket alone is not proof of effectiveness.

Changes to shared skills/tooling go through the same planning and review gates. Restrict edits to authorized project/global scope. Validate with a fresh agent session before adoption. Record the skill version for each run/attempt; existing contexts retain the old instructions until explicitly reloaded. Roll out at a milestone or run boundary by default. An urgent rollout needs coordinated checkpoints and fresh readiness acknowledgements, not editing live instructions and assuming every agent absorbed them.

## Delivery dependency graph

These are independently reviewable increments, not prerequisites for the first Kanban board. Shared migration/CLI registration edits must be integrated serially even when feature work is parallel.

| Increment | Working deliverable | Depends on |
| --- | --- | --- |
| W0 · Contracts | Reviewed command/data contracts for specification revisions, preparation/authorization, validation actors, run readiness, and report decisions. No runtime claim. | This design |
| W1 · Prepared work | Drafts, basic parents/dependencies, reviewed specifications, execution approval, and transactional eligibility including invalidation. CLI behavior and subprocess race tests. | W0 |
| W2 · Independent validation | Explicit project/run QA policy and separate validator decisions/evidence. Preserve legacy human QA and stale-owner protections. | W0 |
| W3 · Problem processing | Retrospective coverage including old-report follow-ups, idempotent decision/ticket links, recurrence, and milestone deferral. Bounded retrospective works without a watcher. | W0; W1 for epic links and deferred eligibility |
| W4 · Single-job worker | Assignment launch payload, one-ticket lifecycle, atomic block/release, durable handoffs, and fresh-session recovery. Work skill taught only supported commands. | W1, W2 |
| W5 · Managed team | Exclusive coordination, persisted run profiles/launch intents, readiness acknowledgements, capacity accounting, and host-specific reconciliation. | W0; W4 for live worker integration |
| W6 · Skills integration | Plan, worker, orchestrator, retrospective, and improver skills with versioned references; updated owned installation/bootstrap and fresh-host validation. | W1-W5 |
| W7 · Observed case study | Approved sample project built by the managed team, independently audited and validated, with workflow-improvement evidence. | W6 and explicit project/scope approval |

After W0, W1/W2 and the report-coverage part of W3 can be implemented in separate worktrees. W5 bookkeeping can proceed alongside W4 after its interfaces are fixed. Allocate workers based on actual independent deliverables and available slots. Each increment must have concrete implementation tickets, targeted tests, and independent review before execution; this table is a delivery graph, not a substitute for prepared tickets.

## Acceptance and case-study observation

Before W7, use deterministic tests plus real fresh-host smoke checks. Validate: draft/unapproved/stale-spec claims rejected even under race; assignments and dependencies respected; new notes do not invalidate approval; worker-created tickets remain draft; duplicate coordinators fenced; lost launch response does not duplicate workers; old contact not treated as death; fresh single-job workers exit; restart preserves scope/claims/cursors; role failure blocks readiness; host capacity includes reviewers/improver; unsupported hosts report limitations. Test no-spawn triage separately and label it as reduced operation.

The proposed W7 project is a local React/MUI reading-list app with add/edit/delete/search/tag bookmarks, a small API with SQLite persistence, API tests, and a browser workflow. No accounts, deployment, scraping, or extension. Project/scope approval is still required. A separate execution proposal can use delegated agent review where authorized; project approval does not waive a retained plan gate.

The observer remains outside implementation and does not quietly fix the app. Inspect supported CLI reads and consistent read-only SQLite snapshots/events at startup, preparation approval, dispatch, handoff, integration, restart, and completion. Never copy a live WAL main file as a complete snapshot. Correlate run/ticket revisions, claims, identities, actual host activity, worktree commits, and test artifacts. Use event history to detect transient violations, not just final states. Keep the observer's resource use in the capacity budget. On the current four-slot host, a separate observer, orchestrator, and improver leave only one execution slot. Use a supported arrangement with additional capacity before claiming simultaneous implementation was validated; otherwise report that part as unvalidated. Do not silently combine the independent observer with the orchestrator or exceed host limits.

Independently exercise the app and run validation against the combined commit. Log interventions and missing evidence; absence of an observed violation is not proof of every transition. Report app correctness and workflow compliance separately. The observer may validate as an explicitly authorized agent, never by impersonating a human or changing records through SQL.

Observe real problems and two labelled controlled drills: a harmless incorrect setup instruction and a low-priority issue for milestone deferral. Verify reporting, grouping, urgency decisions, prepared linked tickets, shared scheduling, independent fix validation, and a fresh worker repeating the affected workflow successfully. Keep seeded drills distinct from naturally occurring failures. A single successful repeat establishes limited evidence, not permanent effectiveness.

## Source and adaptation

[Ivo Kund, Loops, graphs & harnesses](https://www.ivokund.com/loops-graphs-harnesses-getting-quality-out-of-a-software-factory/) motivates separate preparation/execution loops, preserving intent alongside critiques, explicit validation ownership, and learning from failures. The contracts above are Conductor-specific decisions. We do not adopt mandatory PRs, fixed reviewer counts, production actions, or a large factory runtime.
