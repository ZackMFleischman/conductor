# Conductor agent integration: minimal installation

Design contract · 2026-09-12 · Hooks excluded from initial delivery

The [team workflow contract](2026-09-12-conductor-team-workflow.md) defines the approved planning, review, fresh-worker, startup, and improvement-agent behavior. It supersedes the earlier persistent-worker default. Only conductor-work is installed in the tracer; later skills ship with their supporting core operations.

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

**Processing and improving** uses two cooperating skills. conductor-retrospective performs bounded analysis, distinguishes observed facts from inferred causes, groups reports, records coverage, and proposes evidence-backed remedies. conductor-workflow-improver is a managed agent that watches new reports and follow-ups, invokes that analysis when useful, and chooses immediate preparation, milestone deferral, or observation with a revisit trigger.

The improver creates or links draft tickets under a persistent Workflow Improvements epic. The main orchestrator prepares, prioritizes, authorizes, and dispatches those tickets through the shared worker queue. The improver does not spawn its own pool. Independent validation checks both delivery and later effectiveness; closure alone does not prove the workflow improved. Preserve original reports, decisions, later additions, and recurrence evidence.

Initially, bounded retrospectives can use the tracer's problem list, ordinary improvement tickets, and a linked Markdown review. Reliable watching requires persisted coverage, idempotent ticket links, explicit deferral, and restart reconciliation. Do not advertise that runtime until those operations ship. See the team contract for rollout boundaries and the dependency graph.

## Skills versus always-loaded instructions

| Location | Contents |
| --- | --- |
| User-level bootstrap | One small discovery instruction and routing to conductor-work |
| conductor-work skill | Context, explicit sessions, claims, updates, optional checklists, dependencies, linked plans, reporting failures, handoff and QA |
| conductor-plan skill | Investigate, create draft tickets and dependencies, arrange independent critique, and prepare a versioned execution proposal |
| conductor-worker skill | Execute one assigned ticket through conductor-work, checkpoint, release, and end; persistent polling is opt-in |
| conductor-orchestrator skill | Reconcile required roles and approvals at startup; supervise planning, authorization, adaptive dispatch, validation, and integration |
| conductor-workflow-improver skill | Watch scoped reports, triage urgency, invoke retrospective analysis, and feed draft improvement tickets into the shared queue |
| conductor-retrospective skill | Bounded report analysis, coverage, diagnosis, proposed remedies, and effectiveness review |
| Referenced skill files / CLI help | Exact commands and JSON contracts, examples, conflict recovery, troubleshooting |
| Repository instruction files | No Conductor edits required; retain existing project-specific conventions |
| Hooks | None initially; a narrowly scoped optional extension only when supported by actual workflow failures |

Install the skills in supported user-level locations, generated from one source per skill for the two hosts. Only conductor-work is needed for the CLI tracer. Add planning and retrospective skills with their supported operations, a single-job worker after gated claiming and block/release, then the managed orchestrator/improver after readiness and native delegation are validated. Atomic filtered selection is required for the optional persistent polling mode, not for an explicitly assigned single-job worker. Keep skills concise and version their examples with the executable. Do not publish instructions for commands that have not shipped.

Sessions do not depend on hooks. The work skill starts or resumes an explicit session, retains its ID, and supplies it on subsequent CLI calls. Parallel agents use separate IDs even in the same directory. Host session identifiers can be used when available, but a generated ID is the portable baseline. Active claim IDs are persisted and recoverable in SQLite, without credential-cache files. Owner mutations supply session and claim IDs; state changes also supply the expected revision. Preserve request IDs on retry. Check claims and revisions in the application core.

### Identity, worktrees, and delivery handoffs

conductor-work selects an explicit registered agent identity before starting a session. Use the identity supplied by the user or orchestrator; do not identify every Codex or frontend session as the same agent. If none was supplied, register a unique project-scoped name and report it so later assignments can target it. Retain the stable agent UUID and session UUID in the handoff. Several sessions may intentionally share one identity, but each session must acquire its own claim. Registering an identity never proves an agent process is running.

Load assigned_to before claiming and obey it. Unassigned work is available within the user's task scope; another agent's assignment is not. Reassignment requires the explicit authorized coordination action and must wait until an active claim is released. Workers do not change assignment to make their chosen filter pass.

Record actual worktree, branch, and starting commit. Use an isolated worktree for implementation by default through existing Git/host tools; intentional serial or integration work may use the shared checkout. Reuse a safe existing worktree when appropriate. Before recovery, inspect uncommitted changes and the old worker's state; releasing a database claim does not stop that worker or make its files safe to overwrite. Do not discard work or delete a worktree as routine cleanup without confirming its changes are preserved.

Submission includes changed scope, source commit IDs, commands/results, remaining QA, and delivery/integration handoff. Read the project or linked plan's direct-versus-PR preference. A PR may bundle several coherent tickets; do not create one PR per ticket automatically or require PRs in a direct-landing project. Keep ticket acceptance separate from review/merge status. In the tracer record PR links and delivery membership as notes; ship dedicated commands only when implemented.

Use an ordinary integration ticket for combined validation. Accept implementation tickets against their declared worktree-level criteria; do not create a dependency where integration waits for tickets that themselves wait for integration. Integrate branches serially and validate the exact final landed SHA, including after squash/rebase. Record the source commits and target SHA with the results. Before treating that evidence as current, compare the target again; a newer SHA needs new checks. A green branch test or approved PR alone does not establish that combined work functions.

Availability is reported honestly: active claims imply busy; declared idle/stopped, last tracker contact, and host-observed status are separate. Successful session writes update contact; idle workers use explicit session contact when polling. Quiet is not dead. Checkpoint/release before stopping, and never silently revive a stopped session.

## Planning and execution skills

conductor-plan prepares work before implementation. Capture immutable original intent and explicit user amendments, non-goals, measurable criteria, verification method/owner/stage, linked documents, and a dependency graph. Independent reviewers critique substantial plans in fresh contexts; preserve their feedback and version the synthesized proposal. Bound review rounds and escalate unresolved significant findings. Small changes can use an explicitly recorded lightweight policy.

The orchestrator checks preparation and execution authorization before dispatching implementation. Core claim operations enforce those gates against the current specification revision. Progress notes do not invalidate approval; material requirement, interface, dependency, or validation changes do. Worker discoveries enter draft/triage; creating a ticket does not authorize implementation. Preparation approval and result QA are separate policies. Existing human QA cannot be bypassed by an agent claiming to be human; implement explicit independent-agent validation before using it in the case study.

## Single-job worker default

conductor-worker receives an explicit identity, ticket, approved scope, relevant context, delivery policy, and isolated worktree. It uses conductor-work to claim, execute, validate, report problems, and hand off one ticket. It may create linked draft discoveries and record blockers. After submission or an atomic block-and-release handoff, it ends its session. A fresh session handles the next assignment, reconstructing context from tickets and artifacts.

A stable identity may be reused after its prior attempt is reconciled. A session and claim are unique to an attempt; assignment alone grants no ownership. If release is uncertain, pause and reconcile. Quiet is not proof of death. Never overwrite a previous worker's uncommitted changes as routine recovery.

The orchestrator selects concurrency from independent ready work and actual resource capacity, rather than a fixed worker count. Include reviewers, validation agents, the improver, and starting/unknown launches in applicable host limits. Allocate isolated ports/test data as well as worktrees. Release capacity only after a worker is confirmed finished, not merely after submission.

Persistent workers are an explicit alternative when repeated spawning is impractical. They require atomic filtered claim-next, assignment/dependency/approval checks, atomic block/release, retained scope, bounded error handling, and interruptible idle polling (default 30 seconds). Empty polling holds no claim or transaction. They do not broaden scope, restart closed hosts, or become the default. Do not publish example commands before implementation.

## Orchestrator and team startup

conductor-orchestrator is explicitly started for an authorized project/epic outcome. Every start or resume runs the [startup protocol](2026-09-12-conductor-team-workflow.md#startup-and-resumption): check registration/access/capabilities, reconcile exclusive project coordination ownership, restore run scope and approval state, inspect existing children/claims/worktrees, and obtain fresh readiness acknowledgements from required roles. Registered identities and last contact are not liveness evidence.

Ensure the workflow improver is running for an improvement-enabled run. Start planning/review agents as needed; start implementation workers only for prepared, authorized, independent work. Reuse healthy children before launching replacements. Persist a unique launch intent before a host call, count unknown starts against capacity, and reconcile ambiguous results before retrying. Never hold a SQLite transaction open over a host operation. Coordination mutations are fenced against superseded ownership.

The main loop is prepare, authorize, dispatch, independently validate, integrate, and reconcile. Preserve substantive communication in tickets. The improver feeds proposed remedies into that loop; it does not rewrite active shared instructions or expand scope by itself. PR grouping remains independent of tickets, merges are serial, and combined validation records the exact target SHA.

Use native host delegation only where available; no assumed control over unrelated providers. A reduced triage-only mode can record/assign work but must disclose that agents were not launched. Enable reduced operation only under the run's explicit policy; do not silently mark the team ready when a required role is unavailable.

On stop, cease new dispatch, request checkpoints, preserve unresolved ownership and host references, and report any children still active. The next authorized start reconciles that state. No hook, daemon, scheduler, or custom process supervisor is required. The full team contract defines readiness, review/approval invalidation, capacity, and improvement coverage acceptance scenarios.

## Intended installation experience

The tracer already installs the executable, conductor-work, and the owned user-level bootstrap. Follow the [quickstart](../../quickstart.md) for actual setup/init syntax and the current explicit remove/reinstall upgrade procedure. Later skills extend the same installer when their core operations are shipped and validated; do not install design-only command examples.

Preserve user edits, active instruction-file precedence, and host permission policy. Use portable file copies rather than symlinks. Validate paths with spaces on Windows and fresh sessions in each supported host. Skill updates require a documented reload/fresh-context boundary and explicit compatibility checks; installing new bytes does not update an already-running agent's context.

## Delivery and acceptance

The tracer includes the work skill, bootstrap setup, registration, and a short quickstart. Validate that fresh Codex and Claude Code sessions coordinate one ticket across worktrees and append a problem without hooks or per-repository setup. An unrelated repository must stay unaffected.

The next skill increments follow the team contract's dependency graph: agree on core contracts; add prepared-work eligibility and independent validation; add report coverage and deferral; validate single-job workers; add managed run/launch/readiness bookkeeping; then install and validate the combined skills. Independent core slices can be developed in parallel worktrees after interfaces are fixed. A bounded retrospective can be useful before the continuous improver exists.

These increments do not delay the first React/MUI Kanban board and require no portal. Skills ship only after the CLI operations they reference and fresh-host acceptance work. Update only owned bootstrap/skill content, preserving user changes and documenting fresh-session requirements. Test upgrades as well as first installation.

The reading-list case study stays paused until the ideas are implemented and validated and the user approves the project and scope. Its independent observer verifies both product behavior and tracker/agent activity, including the improvement loop, without pretending agent QA is human approval.

There is no mandatory adapter or hook phase. Add an intervention only for an observed failure that a smaller skill/core change cannot address.
