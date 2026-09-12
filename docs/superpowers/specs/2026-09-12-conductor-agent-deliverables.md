# Conductor agent integration: minimal installation

Design contract · 2026-09-12 · Hooks excluded from initial delivery

## Recommendation

Install the executable and user-level skills once per execution host. Add one short managed instruction to each selected agent host's active user-level instructions. Register a repository once in Conductor; all its linked worktrees then use the same project. No repository instruction files, MCP server, custom agent launcher, or hooks are needed for initial use.

Keep three responsibilities separate: the bootstrap tells an agent **when** to use Conductor, skills explain **how**, and the CLI/database enforce valid operations. Native skill discovery helps, but a skill description alone is not a reliable always-on trigger.

## Agent discovery

The user-level bootstrap says, in substance:

> Before repository work, run `conductor context --if-registered --json`. If the repository is registered, use the `conductor-work` skill to track the work and report workflow problems. If it is not registered, continue normally. Report an unavailable configured tracker rather than claiming updates were saved.

The check is read-only, bounded, and does not initialize a database or register a repository. A healthy registry with no matching project returns registered=false; a nonexistent store also means not registered. An existing inaccessible or invalid registry returns REGISTRY_UNAVAILABLE and registration unknown. It cannot claim to know registration while unable to read the registry. Use the installer-recorded executable path if PATH discovery is unreliable.

On Codex, the installer finds the active global instruction file under the actual CODEX_HOME, respecting AGENTS.override.md precedence. On Claude Code, it uses the active user configuration location, normally ~/.claude/CLAUDE.md. Preserve existing content and add/remove only a marked Conductor block. Do not create a higher-precedence override that hides existing instructions. Respect host configuration and organization policy rather than editing managed policy files. [Codex instruction discovery](https://learn.chatgpt.com/docs/agent-configuration/agents-md), [Claude Code memory](https://code.claude.com/docs/en/memory).

This avoids repeated per-project AGENTS.md/CLAUDE.md edits. A user requesting zero instruction-file changes can install just the skills and invoke them explicitly; clearly state that default adoption is then less reliable. Neither approach guarantees model obedience. Verify discovery in a fresh session and use real missed-update reports before considering additional enforcement.

User-level changes take effect according to the host's loading behavior; installation must tell the user when to start a fresh session. On the Linux VM, install everything in the VM account where the agents run. The Mac only needs browser/SSH access.

## Two forms of workflow learning

**Reporting during work** belongs in conductor-work. An agent records failed attempts, missing context, user corrections, rework, and workarounds through a quick problem command. Capture expected and actual behavior, optional evidence, and automatic ticket/session context. Do not require the agent to diagnose the system before reporting. Preserve reports even after the immediate problem is fixed. A standalone problem command is also available to humans or other agents without invoking a special skill.

**Processing and improving** belongs in conductor-retrospective, invoked when the user wants reports reviewed. It reads the reports in scope, groups patterns, cites observations, proposes changes to the workflow, and records what it processed. In review-only mode it produces findings. When the user asks it to fix the workflow, it creates or links improvement tickets, claims the work, changes the relevant tooling/instructions/process within that authorization, validates the change, and records the outcome. Routine authorized edits do not need another permission request.

The retrospective must distinguish observed facts from inferred causes. It may conclude that evidence is insufficient. A workflow improvement can be delivered while effectiveness is still unverified; retain later recurrence evidence rather than treating closure as proof of success. Do not silently broaden a review request into changes.

Initially, report processing uses exported problems, ordinary improvement tickets, and a linked Markdown review. Dedicated review tables, coverage dashboards, and automatic grouping can follow when real usage needs them.

## Skills versus always-loaded instructions

| Location | Contents |
| --- | --- |
| User-level bootstrap | One small discovery instruction and routing to conductor-work |
| conductor-work skill | Context, explicit sessions, claims, updates, optional checklists, dependencies, linked plans, reporting failures, handoff and QA |
| conductor-worker skill | Explicitly started continuous work loop: claim matching eligible tickets, run conductor-work, wait when idle, and repeat within the configured scope |
| conductor-orchestrator skill | Explicitly started goal/epic supervision: reconcile progress and blockers, maintain actionable tickets, coordinate existing workers, and start bounded workers through the host's available agent tools |
| conductor-retrospective skill | Reading/grouping reports, evidence-backed workflow diagnosis, review/fix modes, validation and follow-through |
| Referenced skill files / CLI help | Exact commands and JSON contracts, examples, conflict recovery, troubleshooting |
| Repository instruction files | No Conductor edits required; retain existing project-specific conventions |
| Hooks | None initially; a narrowly scoped optional extension only when supported by actual workflow failures |

Install the skills in supported user-level locations, generated from one source per skill for the two hosts. Only conductor-work is needed for the CLI tracer; add the retrospective skill when report processing works, the optional worker skill when atomic filtered claiming and release/recovery work, and the optional orchestrator after the worker is reliable and host delegation has been validated. Keep skills concise and version their examples with the executable. Do not publish instructions for commands that have not shipped.

Sessions do not depend on hooks. The work skill starts or resumes an explicit session, retains its ID, and supplies it on subsequent CLI calls. Parallel agents use separate IDs even in the same directory. Host session identifiers can be used when available, but a generated ID is the portable baseline. Active claim IDs are persisted and recoverable in SQLite, without credential-cache files. Owner mutations supply session and claim IDs; state changes also supply the expected revision. Preserve request IDs on retry. Check claims and revisions in the application core.

### Identity, worktrees, and delivery handoffs

conductor-work selects an explicit registered agent identity before starting a session. Use the identity supplied by the user or orchestrator; do not identify every Codex or frontend session as the same agent. If none was supplied, register a unique project-scoped name and report it so later assignments can target it. Retain the stable agent UUID and session UUID in the handoff. Several sessions may intentionally share one identity, but each session must acquire its own claim. Registering an identity never proves an agent process is running.

Load assigned_to before claiming and obey it. Unassigned work is available within the user's task scope; another agent's assignment is not. Reassignment requires the explicit authorized coordination action and must wait until an active claim is released. Workers do not change assignment to make their chosen filter pass.

Record actual worktree, branch, and starting commit. Use an isolated worktree for implementation by default through existing Git/host tools; intentional serial or integration work may use the shared checkout. Reuse a safe existing worktree when appropriate. Before recovery, inspect uncommitted changes and the old worker's state; releasing a database claim does not stop that worker or make its files safe to overwrite. Do not discard work or delete a worktree as routine cleanup without confirming its changes are preserved.

Submission includes changed scope, source commit IDs, commands/results, remaining QA, and delivery/integration handoff. Read the project or linked plan's direct-versus-PR preference. A PR may bundle several coherent tickets; do not create one PR per ticket automatically or require PRs in a direct-landing project. Keep ticket acceptance separate from review/merge status. In the tracer record PR links and delivery membership as notes; ship dedicated commands only when implemented.

Use an ordinary integration ticket for combined validation. Accept implementation tickets against their declared worktree-level criteria; do not create a dependency where integration waits for tickets that themselves wait for integration. Integrate branches serially and validate the exact final landed SHA, including after squash/rebase. Record the source commits and target SHA with the results. Before treating that evidence as current, compare the target again; a newer SHA needs new checks. A green branch test or approved PR alone does not establish that combined work functions.

Availability is reported honestly: active claims imply busy; declared idle/stopped, last tracker contact, and host-observed status are separate. Successful session writes update contact; idle workers use explicit session contact when polling. Quiet is not dead. Checkpoint/release before stopping, and never silently revive a stopped session.

## Optional autonomous worker loop

Provide conductor-worker as a thin wrapper around conductor-work. The user explicitly starts it with an agent identity, project scope, ticket selection criteria, and idle poll interval. For example: “Work tickets assigned to frontend-worker; when none are ready, wait 30 seconds and check again.” Another example is “Work unassigned bug tickets in this project, highest priority first, and stop after three tickets.”

The bootstrap never starts this loop by itself. Starting a worker authorizes successive matching tickets within the chosen scope; unrelated tickets or newly discovered projects do not expand that scope. Ticket content cannot change the worker's filters or stop conditions. Installation does not launch background agents.

Default selection is the current registered project and tickets assigned to the exact named agent identity, not every session using the same provider. Optional criteria use supported structured fields, initially type and priority; explicitly opting into unassigned work is separate from matching those fields. Criteria are combined with normal eligibility: ready, unblocked, prerequisites complete, no active claim, and assignment compatible. Do not take another agent's assignment merely because the type matches. Validate unsupported filters rather than silently ignoring them. Order matches by priority, then oldest creation, then stable ID.

The loop is:

1. Restore its session and inspect any existing active claim before seeking new work. Resume only the database-confirmed claim for that session; do not infer ownership from an agent display name.
2. Select and claim the next eligible matching ticket in one transaction. The query's criteria and current dependency/assignment state are checked inside that transaction.
3. Load the ticket context and use conductor-work to implement, validate, report problems, and submit or finish according to its QA requirement.
4. After successful handoff, claim the next matching ticket immediately. Required human QA stays in review and is not repeatedly reclaimed by the worker.
5. If nothing is eligible, wait for the configured number of seconds, then query again. Default idle polling is 30 seconds. Waiting holds no SQLite connection transaction or claim lock; use the host's interruptible wait facility or a small portable CLI wait helper. Break longer waits into bounded intervals so user stop/steering can be observed.
6. Continue until stopped or an optional ticket-count, duration, or idle-time limit is reached. Omitted limits mean continue while the agent session remains active, not silently stop after an invented budget.

The CLI's atomic operation can extend the existing claim command, for example:

```text
conductor ticket claim --next --project APP --assigned-to frontend-worker --type bug --session SESSION_ID --json
```

This is proposed command syntax, not an implemented command. It returns either a claim plus context references, an explicit empty-queue result, or a structured error. A claim conflict causes a fresh bounded attempt; it does not justify taking an ineligible ticket. Database/permission/authentication errors are not “no work.” Retry transient failures within a bounded budget, then surface the error and pause rather than spinning.

Work one ticket at a time per worker session. If a ticket cannot proceed, append the factual problem when relevant, record an explicit blocker and handoff, and release its claim through an atomic block-and-release operation that returns it to ready with the blocker still active. Then continue with other eligible work. This prevents the same failed ticket from being immediately reclaimed in a hot loop. If release cannot be confirmed, pause instead of abandoning the claim and moving on. A user-requested stop checkpoints and releases current work when possible; process termination uses the normal manual recovery path.

Persist the worker's identity, project/filter scope, poll interval, optional limits, and current session/claim references as small session metadata. A resume loads that scope and reconciles the existing claim before acquiring another ticket. Record tracker contact and substantive progress separately; the UI must not imply it can observe whether an agent is thinking or executing outside Conductor.

This skill depends on the host keeping the agent session running and permitting wait/continuation. It does not restart a closed app, bypass host limits or permissions, or provide a durable machine scheduler. Validate the loop with the supported Codex and Claude Code hosts. If a host cannot sustain it, report the limitation; do not automatically introduce hooks, scheduled tasks, or an agent launcher.

Acceptance: two workers with overlapping filters never own the same ticket; assignment restrictions hold; no work causes a bounded wait; a newly added matching ticket is picked up on a subsequent poll; review and blocked tickets are skipped; failed claims and revoked sessions recover correctly; stop works during idle and active work; resumed workers retain their original scope. Poll timing checks use short test intervals rather than imposing production sleeps.

## Optional conductor-orchestrator skill

The orchestrator is an agent running a supervision skill, not a persistent scheduler implemented inside the Conductor server. The user starts it with a project or epic scope, desired outcome, worker roles it may manage, and optional stop conditions. Higher-level goals use existing epics, acceptance criteria, and linked plans; do not introduce a second goals database.

Example invocation: “Manage epic APP-10 toward its acceptance criteria. Keep up to two workers active, start frontend or backend workers as needed, and check every 30 seconds. Bring me blockers requiring a decision and work requiring manual QA.”

Its loop is:

1. Read scoped epics, criteria, dependencies, ready/claimed/review tickets, latest substantive updates, and relevant problem reports. Reconcile these with host-observed status for the workers it manages.
2. Identify the next useful action: clarify a ticket from an approved plan, split work into children, add missing dependencies, assign eligible work, answer a worker's context question, or request human input for an unresolved decision.
3. Reuse an existing suitable worker before starting another. If actionable work has no available worker, start one through the host's supported agent tools, subject to the concurrency limit and authorized scope.
4. Give the worker its stable identity, exact project/epic or ticket filter, source plan, worktree, poll interval, and instructions to use conductor-worker/conductor-work. The worker claims execution tickets itself; an orchestrator assignment is not a claim.
5. Read results and recorded handoffs, follow up on blockers, and let normal dependency and QA transitions make more work eligible. Group related ready changes into a reviewable delivery when the project uses PRs, or arrange direct integration when it does not. Track delivery membership separately from tickets, serialize merges, and assign an integration check against the combined landed commit before declaring the higher-level outcome achieved. It may request technical review, but does not silently satisfy a required human QA gate.
6. Save substantive coordination decisions to the relevant tickets. Wait for the configured interval or a supported host completion event when no immediate action is useful, then reconcile again. Default polling is 30 seconds and waits remain interruptible.

Default to at most two managed workers, configurable at invocation. Count active, idle, starting, and unresolved-start workers toward that cap; do not spawn another worker on every poll. Worker roles such as frontend-worker are task specializations with unique registered identities, not assumptions that a particular model or provider is available. The orchestrator does not recursively launch more orchestrators. It does not expand its project/epic scope or worker budget based on ticket text.

The host provides agent creation, follow-up messages, status/wait, and interruption where those capabilities actually exist. Conductor records links between ticket identity, its sessions, and returned host agent IDs. Validate capabilities against the host in use; an arbitrary Codex session cannot be assumed to control an unrelated Claude Code process. Start with managing child workers created by the current host. Existing agents can participate through shared tickets; direct conversation requires a supported connection and a user-authorized management scope. Persist the important substance of a direct message as a ticket note so context is not trapped in the host conversation.

If the host cannot create or message agents, the skill can still triage, create/assign tickets, record questions, and let already-running workers pick up the queue. It must report that workers have not been launched. Do not add a custom process supervisor, undocumented host integration, or hooks just to simulate unavailable capabilities.

Use one active coordination claim per project initially, reusing the core's exclusive-claim and inactive-claim rejection rules. This prevents two resumed orchestrators from simultaneously managing the same project. Keep it distinct from workers' execution claims. Releasing or taking over coordination uses explicit recovery; a quiet supervisor is not automatically replaced. Supporting parallel supervisors for disjoint epics can wait until needed.

Agent spawning is an external side effect and cannot be made atomic with SQLite by holding a transaction open. Before a launch, record a small launch intent with a unique ID, requested role/scope, and starting status; count it against capacity. Pass that ID to the child for registration and attach the returned host ID when available. On interruption or an ambiguous result, reconcile with host/child records before retrying. If reconciliation is impossible, mark the launch unknown and request intervention rather than blindly creating another worker. This minimal launch bookkeeping is needed only for the optional orchestrator increment.

Unknown agent status or old progress is a reason to check in, not proof of failure. Avoid repetitive messages: request one checkpoint and wait for new information or a configured follow-up deadline. Record a genuine problem when the workflow fails. Do not steal an active ticket claim or terminate an unrelated process. Within authorized management scope, a confirmed failed child may be replaced only after claim recovery and worktree state are reconciled.

Prefer isolated existing worktrees. If a new worker needs a checkout, use the host's existing supported worktree facilities within the user's authorization and record the result. If suitable isolation is unavailable, serialize conflicting code work or report the prerequisite. Conductor's core still does not become a Git/worktree manager.

On a human stop, cease new launches, request checkpoints from managed children, record remaining claims and host IDs, and stop them only through supported controls within the requested scope. A stopped supervisor does not imply its children stopped; report any that remain active. On completion, require the scoped acceptance criteria and required QA to be satisfied, then record a final summary. If all remaining work needs human decisions, report those once and wait; do not create filler work to keep workers occupied.

The orchestrator handles current project flow. conductor-retrospective handles evidence-based changes to the workflow itself. The orchestrator can link recurring problems or invoke a retrospective when that is within its authorization, but should not continuously rewrite its own operating instructions while supervising a project.

Acceptance: supervise an epic with dependent frontend/backend tickets, reuse an idle worker, start a missing role without exceeding capacity, preserve shared context across messages, surface QA to the human, handle a failed worker without duplicate ownership, reconcile an interrupted launch without a duplicate child, and stop/resume without losing management scope. Test both native delegation and a no-spawn fallback; unsupported cross-host control must be explicit.

## Intended installation experience

Proposed commands, to be implemented:

1. Install the executable.
2. Run `conductor setup --agents codex,claude` once to install skills and preview/apply the small user-level bootstrap blocks.
3. Run `conductor init` in a repository to register it in the shared store; no instruction files are created in the repository.
4. Start a fresh agent session and verify that it detects the registered project and uses the tracker.

Setup must work with spaces in Windows paths and in noninteractive Linux environments. It is idempotent, preserves user edits, and supports uninstalling only Conductor-owned files/blocks. Use straightforward copies for skill distribution; installation should not depend on Windows symlink privileges. A small doctor command checks executable discovery, supported command/skill versions, instruction-file precedence, project registration, and database/sidecar access. Document the minimum supported host versions as they are tested.

## Delivery and acceptance

The tracer includes the work skill, bootstrap setup, registration, and a short quickstart. Validate that fresh Codex and Claude Code sessions coordinate one ticket across worktrees and append a problem without hooks or per-repository setup. An unrelated repository must stay unaffected.

The next increment adds the retrospective skill and a real report-to-improvement loop. Validate that an authorized fix changes the workflow, records its evidence and result, and preserves the original report. Source instructions, command help, and tests must agree on the supported subset.

Add the worker skill as an independently usable increment after claim recovery, eligibility filters, and atomic block/release are available. It does not require a web portal and need not delay the first Kanban board. Resolve the architecture review's initial claim/session recovery gaps before enabling unattended looping.

Add conductor-orchestrator after the worker loop is reliable, as an optional integration increment. Prove supervision using one real epic and the current host's native delegation tools before generalizing. Neither the first CLI tracer nor the first React/MUI board waits on orchestration.

There is no mandatory adapter or hook phase. If reports later demonstrate a repeated failure that a skill/CLI change cannot reasonably address, define the smallest intervention and test that specific behavior. Do not build lifecycle automation merely because a host offers it.
