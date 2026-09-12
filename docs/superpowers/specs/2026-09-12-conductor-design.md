# Conductor: local tickets for coding agents

Design proposal · 2026-09-12 · One execution host; Windows local or Linux VM with browser access over SSH

The [team workflow contract](2026-09-12-conductor-team-workflow.md) updates this target design with planning/review gates, fresh single-ticket workers, adaptive concurrency, managed startup, and continuous workflow improvement. The CLI tracer is implemented; these later operations are not yet installed.

## Delivery priority

Build a working end-to-end CLI tracer first, then iterate into the remaining features. The detailed sections describe the target product, not a requirement to build the whole schema or portal before first use. The delivery sequence at the end is authoritative: real agent coordination first; a small Kanban board layered over the same core next; Markdown rendering, document sidebars, and richer analysis views afterward. Hooks are not an initial deliverable and are only considered when a demonstrated workflow failure requires one.

## Recommendation

Build one executable that provides a ticket CLI and a local web dashboard, backed by one SQLite database outside Git checkouts. All interfaces use the same application library and transaction rules. Agents can work without the dashboard running.

Conductor records work, ownership, decisions, dependencies, handoffs, and workflow problems encountered along the way. The core works with agents launched through the user's existing tools. An optional, explicitly started conductor-orchestrator skill may later use the current host's agent tools to coordinate or launch workers. Assigning a ticket by itself does not launch or message an agent.

The first installed milestone is a usable CLI tracer: two agents share a project across worktrees, safely claim/update tickets, report problems, and complete or recover a handoff. Dependencies and browser QA are subsequent milestones. The [CLI tracer specification](2026-09-12-conductor-tracer.md) defines the first increment and resolves the architecture review's recovery, transition, and storage-access contract gaps.

## Alternatives considered

| Approach | Benefit | Cost | Decision |
| --- | --- | --- | --- |
| Shared SQLite, CLI and optional web process | Small installation; works without a service; transactions handle concurrent clients | Every writer must share the application rules; agent sandboxes need access to the data directory | Recommended for one computer |
| Local API service owns SQLite; clients use HTTP | One writer interface; easier future remote access | Service startup and availability become prerequisites for every agent operation | Revisit if sandbox access or remote agents become the dominant requirement |
| Integrate a larger existing tracker | Less ticket UI to build | More concepts, deployment, and integration surface to carry | Does not match the requested small initial scope |

This is a design tradeoff, not a feature audit of Beads or Paperclip.

## Architecture and packaging

```text
Codex / Claude Code / other shell-capable agents
                 |
          conductor CLI          Browser
                 |                  |
                 |        conductor serve (loopback)
                 |                  |
                 +-- shared core ---+
                          |
                 SQLite + artifacts
                 outside worktrees
```

Use Go, a pinned pure-Go SQLite driver, and the standard HTTP server for the core and CLI. Build the portal with React, TypeScript, and MUI (Material UI); compile its assets during development/release and embed the build in the Go executable. The first portal is a Kanban board with two-second polling while visible, built after the CLI is useful. Windows is the immediate development target; Linux VM hosting is a supported release target, with a Mac browser accessing it through an SSH tunnel. Build and smoke-test Windows and Linux from the same source before claiming platform support. Native macOS execution can follow; viewing the portal on a Mac requires no Conductor installation there. Frontend development uses Node tooling, but release users need no Node, Python, Docker, database installation, or frontend build step. Exact binary size and supported OS versions must be measured during packaging.

Keep four internal boundaries: storage/migrations, ticket operations, CLI, and web. Any future necessary hook adapter invokes the CLI. No separate rules engine, event bus, or second backend implementation. Introduce storage tables and migrations only as their usable feature slices require them.

Default Windows data location: `%LOCALAPPDATA%\Conductor\`. On Linux use `$XDG_DATA_HOME/conductor`, falling back to `~/.local/share/conductor`. Use native per-user application data locations on other systems. `CONDUCTOR_HOME` explicitly overrides the location. Store `conductor.db`, managed artifact copies, and small session/config files there. WAL mode also creates SQLite sidecar files; backups must account for this.

The database supports multiple projects in one dashboard. Tickets have opaque stable IDs plus human-readable project keys such as `APP-42`.

## Linux VM and SSH access

In the remote-viewing setup, the agents, CLI, Git worktrees, linked files, database, and web process all live on the Linux VM under the user's account. The Mac runs a browser and an SSH tunnel. This keeps the same one-host storage/transaction model; the SQLite file is never shared over SSHFS or synchronized between computers.

Proposed startup on the VM:

```text
conductor serve --host 127.0.0.1 --port 7432
```

Forward the port from the Mac using OpenSSH:

```text
ssh -N -L 127.0.0.1:7432:127.0.0.1:7432 user@work-vm
```

Then open `http://127.0.0.1:7432` on the Mac and authenticate using the portal's local token flow. The tunnel forwards browser HTTP traffic to the VM's loopback listener. The server serves Markdown, images, and downloads from the VM through that same origin; it must never send browser fetches to VM filesystem paths or rely on files existing on the Mac.

Use relative API and asset URLs. Permit an explicitly configured browser origin when the local forwarded port differs from the server port, while retaining Host/Origin validation and CSRF protection. Do not enable wildcard origins for tunneling. On a headless VM, print connection instructions instead of attempting to open a browser. Document running the web process under a user service or an existing persistent terminal so it can survive SSH disconnection; reconnecting the tunnel restores the UI without affecting agent work. A disconnected dashboard displays its stale connection state instead of presenting old data as current.

Windows local use and Linux VM use are independent installations unless explicitly migrated. Cross-host agent writes and database synchronization remain outside version one. SSH forwarding requires that the VM's SSH configuration permits it. No public listener or reverse proxy is necessary for this setup. [OpenSSH forwarding documentation](https://man.openbsd.org/ssh#L)

## Projects and worktrees

Register each Git repository by its canonical absolute `git rev-parse --git-common-dir` result. Linked worktrees resolve to the same project; record each worktree root, branch, and observed commit separately. Normalize paths with platform-aware filesystem behavior, including Windows casing and symlinks.

Do not identify projects by branch, current directory name, or remote URL. Different local clones are separate projects until explicitly linked by the user. Repository moves need an explicit rebind operation. Non-Git folders can be registered explicitly.

Agent commands resolve their project from the current worktree, or accept `--project` outside a checkout. Nothing creates a new database merely because an agent changes directory. Instructions may be committed to the repository; operational ticket state is never committed or merged through Git.

Onboarding must check whether each agent sandbox can read and write the database directory and SQLite sidecars. `conductor doctor` reports the exact missing access. Configure narrowly scoped access through the host's supported settings; do not disable sandboxing. WSL and host-native Windows clients sharing one database are outside the initial supported configuration: use clients in the same OS environment first.

## Ticket model

Each ticket contains a title, type, Markdown description, acceptance criteria, priority, optional parent ticket, optional intended assignee, workflow state, and a monotonic revision. Initial types are `task` (default), `bug`, `feature`, and `epic`. All types use the same core record and interfaces; types do not require separate forms or workflow engines. Keep descriptions bounded and put large evidence in artifacts.

Epics group child tickets and can link the shared design or implementation plan. Show their child statuses and counts; do not invent completion percentages or automatically mark an epic done when its children finish. Epics are excluded from the automatic work-claim queue, and agents create concrete child tickets for execution. Non-epic tickets can also have children when useful. Type and parentage are independent of dependency edges.

Description, acceptance criteria, progress notes, QA instructions, and problem/review narratives support Markdown. Structured checklist items remain optional and distinct from Markdown task-list syntax: checkboxes embedded in a document render as document content and do not mutate ticket checklist state.

Tickets may optionally have a checklist. Agents add one when discrete steps or verification items help track the work; creating, claiming, submitting, and completing a ticket never require a checklist. Simple tickets can use just their description and acceptance criteria. Each checklist item has a stable ID, text, completion state, revision, and completion actor/timestamp. Update individual items transactionally with revision checks, recording changes in the event history. Completing a checklist does not automatically complete the ticket or satisfy required human QA. Work needing separate ownership, dependencies, or review belongs in a linked child ticket. Checklist items are advisory tracking aids, not an additional completion gate.

Workflow states are `backlog`, `ready`, `in_progress`, `review`, `done`, and `cancelled`.

Blocking is an overlay, not another mutually exclusive state: unresolved prerequisites and explicit blocker notes make a ticket blocked. This preserves distinctions such as “review awaiting credentials” and “ready but waiting on another ticket.” The dashboard presents a dedicated blocked filter.

The claimable queue contains only ready tickets with no active claim, no explicit blocker, all required prerequisites done, and any configured deferral condition satisfied. For projects using prepared-work policy, the current specification must also be prepared and authorized; explicit and automatic claims enforce the same gates transactionally. Assigned tickets are claimable only by the matching registered agent identity; unassigned tickets are available to any agent. A resumed agent uses its current claim instead of claiming again.

Parent relationships describe scope. Dependency edges describe execution order. Parents do not automatically block children. Reject cycles in each relationship graph. An explicit `A depends on B` edge means B must be done before A can be claimed or completed. Cancelled prerequisites remain unsatisfied until the dependency is removed or replaced. Start with dependencies inside one project.

New prerequisites can block work already underway. The system records the change, preserves the current claim, and prevents completion while blocked. Reopening a prerequisite flags unfinished dependents; completed tickets are not automatically reopened, but an attention event records affected completed dependents.

Use a configurable project default for manual QA, copied onto each new ticket; initially default it on. Agents submit a completion summary, validation evidence, and QA instructions into `review`, releasing their claim atomically. Human acceptance changes the ticket to `done`; rejection returns it to `ready` with a reason. Tickets explicitly created without manual QA follow their configured validation policy. The later independent-agent policy requires a separate validator identity and fresh context with no implementation participation, plus recorded evidence and acceptance; workers cannot self-accept by starting another session. Simple direct-finish policy remains a separate explicit choice. Agents cannot use routine completion commands to bypass an existing QA requirement.

Done means the acceptance criteria have been accepted. If merge or deployment is required, say so in those criteria and record supporting links. Conductor does not infer that a committed or merged change works correctly.

## Delivery, pull requests, and integration

Tickets describe work and its acceptance. A delivery groups changes that will land together; a PR is an optional review mechanism attached to that delivery. A project chooses direct landing or PR-based landing as its default, with explicit exceptions recorded where needed. No PR is required merely to use Conductor. Do not turn PR status into additional ticket workflow states.

A delivery may include several related tickets. A ticket may link to more than one delivery when work is split or corrected; record what portion each delivers, so a single merged PR cannot silently complete the whole ticket. Initially use ordinary ticket notes and a linked integration ticket for this information. Introduce structured deliveries and ticket-delivery links as a separate coordination increment; neither the tracer nor the first Kanban board needs GitHub API integration.

The later delivery record contains an ID, title, included tickets and scope, direct/PR mode, source and target refs, proposed source commit, optional provider/repository/PR URL and number, observed review/merge status with checked_at, actual landed commit, and linked integration evidence. Support branch refs as navigation aids, but tie evidence to immutable commit IDs. Distinguish locally recorded status from a freshly verified hosting-provider result. PR approval, merge, ticket QA acceptance, and integration success are separate facts. Closed-unmerged PRs and failed checks remain visible; a replacement PR does not erase history.

Group work by a coherent reviewable outcome, a compatible base, and similar readiness. Avoid batching unrelated changes or holding a small ready fix for an unfinished epic. No fixed ticket count or line-count quota: the orchestrator proposes a size that a reviewer can understand and validate. Sequential tickets that intentionally share one PR normally share a delivery branch, with one writer at a time; independent agents use separate worktrees/branches and integrate into the delivery branch serially.

By default an implementation ticket's criteria end at a validated worktree commit and handoff; an ordinary integration ticket owns combining the changes and checking the result. When dependencies ship, the integration ticket depends on those implementation tickets, and downstream work that requires the combined code depends on integration. If a ticket instead explicitly requires landing before it can be accepted, do not make that landing operation depend on the ticket being done. This avoids a circular completion gate.

Integration records included source commits, the exact combined target SHA tested, commands, results, and unresolved failures. Re-check the actual landed commit after merge, squash, or rebase; source-branch checks alone do not verify the final tree. A historical green result stays valid for its recorded commit but cannot authorize a newer tip. Reconcile target HEAD when submitting or accepting integration; show a mismatch rather than implying continuous Git monitoring. Changed targets need checks appropriate to the final combined result.

Skills use the project's existing Git and hosting tools to prepare branches and PRs within user authorization. Conductor records the relationships and evidence; it does not become a Git manager, mandatory PR gate, automatic merger, or CI replacement. PR creation does not authorize merging or deployment. Initial installations can use direct landing with the same integration discipline.

## Ownership and transaction rules

An intended assignee is separate from the session currently holding the work claim. Store stable agent identity, provider, unique session ID, optional parent session ID, worktree, branch, and optional conversation URL. Parallel subagents need separate session IDs.

Agent identities have stable UUIDs and project-scoped unique names; roles are metadata, not identity. Each session explicitly binds to an existing identity, and any session of that identity can claim its assigned work. Assignment changes require a revision check and are rejected while a claim is active. Derive busy from claims and display declared idle/stopped, last contact, and last substantive progress separately. A registered identity or old idle session is not proof of an available process; orchestration reconciles host status before reuse. Stopping requires explicit release first and never silently removes ownership.

Every mutation uses a short transaction. Enable WAL, foreign keys on every connection, a bounded busy timeout, and durable synchronization. Never run network calls, Git commands, or artifact copying inside a database transaction.

Claiming a ticket uses `BEGIN IMMEDIATE`: validate eligibility, insert the exclusive active claim, change state, increment revision, and append an event, then commit. A unique constraint on active claims guarantees a single winner. Two agents racing for the same ticket receive one success and one structured conflict. `claim --next` selects and claims inside that same transaction.

Claims return an opaque, never-reused claim ID as an ownership fence. Owner-only operations require the active claim ID and session. All claim/session state is stored in SQLite; there is no separate credential cache or secret token recovery path. An inactive claim remains invalid, and a resumed session can retrieve its active claim from the database. This is cooperative coordination within one OS account, not authentication against that account. Description/state edits also require an expected revision; stale edits return the current revision and require a refetch. Appended notes do not replace anyone else's notes. Each successful business mutation and its event commit together.

Retried create, claim, update, and completion requests carry an idempotency key. Persist the key, payload hash, and result in the same transaction. A repeated key with the same request returns its original result; reuse with different input fails. Clients retry transient database contention within a small total budget; they do not blindly retry semantic conflicts.

Dependency cycle checks and insertion occur under one write transaction so simultaneous opposite edges cannot both succeed. Parent-cycle checks follow the same rule. Database constraints supplement application validation.

Avoid automatic time-based reassignment in version one. Track last observed activity and last substantive progress separately. After a configurable interval, show “quiet / needs attention,” not “dead.” A long test or model response is not proof that an agent stopped.

A human can explicitly release a claim with a reason in the first tracer. Release atomically invalidates the old claim and returns the ticket to ready; direct takeover assignment can follow later. An old agent's next ticket mutation fails with `CLAIM_REVOKED`; its skill tells it to stop and reload context.

This prevents conflicting ticket ownership, not arbitrary concurrent filesystem writes. Separate worktrees remain the code-isolation mechanism. Takeover cannot terminate an old process; the UI must explain this at that action and show the old session/worktree so the user can stop it. Optional declared file areas provide overlap warnings only, deferred until actual usage shows a need.

## Minimal persistence

| Entity | Stored information |
| --- | --- |
| projects / worktrees | Identity, canonical paths, project defaults, last observed Git metadata |
| agents / sessions | Provider, stable agent label, session identity, parent session, location, last seen |
| tickets | Type, Markdown content, criteria, parent, state, priority, intended assignee, QA requirement, revision |
| checklist_items | Optional per-ticket items with stable IDs, ordering, text, completion state, revision, and completion actor/timestamp |
| dependencies | Unique prerequisite edges |
| claims | Unique claim ID, session ownership, acquisition/release metadata; one active per ticket and worker session |
| blockers | Reason, author, creation and resolution metadata |
| events | Ordered append-only business changes and notes, actor/session, ticket, timestamp, payload |
| workflow_problems | Immutable initial observations, project/session/ticket context, expected and actual behavior; follow-ups and corrections append through events |
| workflow_reviews | Review scope and event watermark, cited problem entries, report artifact, linked improvement tickets; later review updates append through events |
| artifacts / artifact_links | Reusable artifact records with ticket/problem/review associations, relationship (design, plan, evidence, reference), label, URL or managed/live file reference, root/worktree identity, relative path, content hash and observed commit provenance |
| deliveries / ticket_deliveries | Later optional landing groups, included ticket scopes, direct/PR mode, refs and commits, observed PR state, and integration evidence links |
| requests / schema migrations | Idempotency results and supported schema version |

Current-state tables drive queries; events provide a readable audit trail. This is not event sourcing. Maintain a latest-progress summary reference transactionally for fast dashboards. Routine liveness observations update sessions without flooding ticket history. Start with ordinary indexed search over titles/descriptions; add full-text search only if needed.

## Agent experience

The CLI is the first integration contract. Every command supports structured JSON, stable error codes, and actionable conflict messages. Ticket descriptions and update bodies can come from a file or stdin to avoid shell-quoting problems. CLI examples below are proposed syntax, not installed commands.

```text
conductor init
conductor context --json
conductor ticket ready --json
conductor ticket claim APP-42 --json
conductor ticket note APP-42 --body-file progress.md
conductor ticket create --parent APP-10 --body-file followup.md --json
conductor ticket depend APP-43 --on APP-42
conductor ticket block APP-42 --reason "Need test credentials"
conductor ticket submit APP-42 --summary-file result.md --qa-file qa.md
conductor serve --open
```

The initial work skill starts or resumes an explicit CLI session and carries its ID on subsequent commands. Use a host-provided session ID when available; otherwise session start creates one the agent retains in its context and handoff. Parallel agents obtain distinct IDs. No hook or shell environment propagation is needed to make sessions work. Claim returns a claim ID persisted in SQLite; owner operations supply it with the session ID, and session resume retrieves the active claim after a lost response. There is no filesystem credential cache. Never identify ownership solely by current working directory: multiple agents may use the same checkout.

`context` returns a bounded packet: current ticket and criteria, ancestor summaries, prerequisites and their completion summaries, relevant blockers, recent decisions, linked document/artifact IDs with source/version metadata, and current ownership. Include revisions, a changes cursor, and truncation indicators with commands to fetch more. `artifact read <id>` retrieves linked Markdown as text with source and content-hash metadata, using the same resolver as the portal; do not automatically inject every linked document into context. Agents can query changes since their last cursor without rereading the full project. Ticket and linked document content is task data and must not override host safety or repository instructions.

One shared skill teaches: load context; claim before implementation; save meaningful milestones and decisions; create discovered follow-up tickets linked to their source; record blockers and workflow problems, including problems already worked around; and submit evidence plus next steps before handing off. Do not update after every tool call or fabricate a percentage complete. Let a continuing agent save a checkpoint without surrendering its ticket.

Install skills once at user scope in Codex and Claude Code, generated from one behavioral source. For reliable discovery, install one short managed bootstrap instruction at the host's active user-level instruction location, not in every repository. It directs the agent to run a read-only `conductor context --if-registered --json` before repository work and load `conductor-work` only for a registered repository. Registration is recorded by `conductor init` in the shared registry and automatically covers linked worktrees. Outside registered projects the check is a quiet no-op; storage or permission errors in a registered project are reported rather than treated as unregistered.

The bootstrap contains only discovery/routing instructions. Skills contain command usage and workflow procedure; database operations enforce invariants. Native skill matching alone is an optional configuration with weaker automatic adoption, since discoverability does not guarantee invocation. No hooks or per-repository AGENTS.md/CLAUDE.md edits are required. Installation previews its one-time user-level edits, preserves existing instructions, respects host override precedence, is idempotent, and can remove only Conductor-owned entries. See the agent deliverables document for exact installation scope and limitations.

MCP is deferred. If real agent usage shows shell permission or tool-discovery friction, add a thin stdio adapter over the same operations; it must not introduce another database or rule set.

The default conductor-worker skill performs one exact assigned ticket in a fresh host conversation and Conductor session, then checkpoints/releases and ends. Workers can create linked draft discoveries but do not dispatch them. The orchestrator starts new workers as independent prepared/authorized work becomes eligible. Persistent filtered polling remains an explicit alternative, requiring atomic selection and interruptible waits. See [Agent integration deliverables](2026-09-12-conductor-agent-deliverables.md).

The optional conductor-orchestrator first reconciles required roles and plan approval state, then runs preparation, authorization, adaptive dispatch, independent validation, and integration. It restores the workflow improver as a required role for improvement-enabled runs. Concurrency follows actual independent work and host/resource limits, including reviewers and the improver, rather than a fixed worker count. Native host tools handle agent execution; the core stores exclusive coordination, launch intents, readiness, and recovery state. The database and portal do not run an agent scheduler.

## Workflow problem log and retrospective review

Make “this was a problem” a first-class append-only log in the same database, available across projects and worktrees. Its purpose is to expose where the workflow failed to give agents the context, instructions, tools, or coordination they needed. Logging a problem must be faster than creating a fully specified improvement ticket.

Agents record failed workflow attempts, mistaken assumptions, user corrections, repeated work, missing context, confusing instructions, failed handoffs, environment/setup failures, and coordination mistakes when they notice them. Record an incident even if the agent immediately fixes or works around it. No root-cause analysis or proposed solution is required before logging. An intentionally failing test is not itself a workflow incident; discovering that the wrong tests were run or their result was misreported is.

Each entry records:

- A short factual summary of the problem.
- What the agent tried, what it expected, and what actually happened.
- Automatic project, session/agent, worktree, and timestamp context; ticket association is optional.
- Optional evidence references, observed impact/rework, suspected cause, and workaround or correction.

Keep observations separate from hypotheses. For example: “Started APP-43 before APP-42's interface was settled; rewrote the caller after the interface changed” is the observation. “A missing dependency allowed premature work” is a hypothesis for the later reviewer. Do not invent time estimates or attribute the problem to an agent's character or intent.

Suggested CLI, with structured bodies accepted through files or stdin:

```text
conductor problem add --ticket APP-43 --body-file problem.json --json
conductor problem append PRB-17 --body-file correction.json --json
conductor problem list --unreviewed --json
conductor problem export --all-projects --since-event 1200 --json
conductor review save --body-file retrospective.json --json
```

The initial report is immutable through normal interfaces. Workarounds, later evidence, corrections, duplicate links, and review dispositions append with their own author and timestamp; never rewrite the original account. A mistaken report can be explicitly corrected or retracted while preserving its history. Routine recovery does not erase a problem or mark it reviewed. Reports do not require ownership of the affected ticket, so a reviewer, user, or revoked session can still report a coordination failure.

Use the existing transaction, event, and idempotency mechanisms for problem reports and reviews. Retrying one report does not create another occurrence. Distinct real occurrences remain separate even when their summaries match; a later reviewer can group them. An export uses a consistent database snapshot and supplies an upper event watermark, stable IDs, and pagination so a review can cover a large log without silently skipping entries.

The learning loop has two forms: (1) everyday reporting, built into the work skill and `problem` commands, and (2) a manually invoked `conductor-retrospective` skill that processes reports and improves the workflow. The latter reads a selected project or all projects, groups related observations, cites original entries, distinguishes symptoms from supported causes, and asks what workflow change could prevent recurrence. Its report includes patterns, evidence and counterexamples, confidence, proposed improvements, and how each improvement would be checked. “Insufficient evidence” and “isolated occurrence” are valid conclusions; examining workflow causes does not make every failure proof of a workflow defect.

Save each retrospective as an artifact plus structured review metadata identifying the exact problem entries and event watermark it examined. New reports and later additions to reviewed problems surface as unreviewed changes. Concurrent reviews may overlap and remain attributable; no global destructive “consume log” operation. A review can link multiple incidents to one existing or newly proposed improvement ticket, rather than producing one ticket per incident.

The retrospective skill performs bounded review and saves findings and recommendations. Authorized remedies become prepared improvement tickets executed by ordinary workers. An explicit one-off fix request may compose analysis, preparation, and execution within its existing authorization. Do not ask again for routine edits already authorized by a fix request. Fix mode is not blanket permission for unrelated changes or bypassing user/project constraints. Track adoption through linked tickets and append verification or recurrence evidence. A completed improvement ticket means the change was delivered; effectiveness remains unverified until supported by subsequent observations. An optional user-started managed workflow improver watches reports and follow-ups, invokes bounded retrospective analysis, and submits draft remedies under a Workflow Improvements epic. The orchestrator prepares and dispatches them through the shared queue. No model API, service, or durable scheduler is needed. See the team workflow contract for coverage, deferral, restart, and adoption rules.

The shared work skill instructs agents to log problems when noticed and check for unrecorded corrections at checkpoints and handoff. This starts without hooks; observed adoption problems can inform a later narrowly justified intervention. Automatic candidate reports, if enabled later, must be labeled as such and remain distinct from agent explanations. Do not ingest whole transcripts or raw command output by default; save concise evidence with sensitive values omitted. If storage is unavailable, report the unsaved observation to the user and retry when available; never claim it was recorded.

## Hooks: evidence-driven extension only

Do not build or install hooks initially. Use the bootstrap instruction, skills, and explicit CLI sessions first. Only propose a specific hook when actual problem reports demonstrate that these mechanisms cannot adequately address a recurring omission. Define the failure it addresses, why a simpler skill/CLI change is insufficient, the covered host events, and how success will be checked. Hook work must earn its scope rather than being bundled as preventive infrastructure.

Any later hook is optional and narrowly scoped, must handle the actual installed host's payloads, and must not create continuation loops, fabricate problems, automatically complete tickets, release claims on a normal stop, or alter permission policy. Data integrity always stays in the shared core. Current platform hook documentation establishes feasibility only, not a commitment to implement adapters.

## Dashboard

The first portal opens to a Kanban board: Ready, In progress, Needs QA, and Done. Backlog/cancelled tickets are available through a filter or additional column once needed. Blocking remains a visible badge on the ticket's current column, not an independently persisted workflow state. Cards show ticket type/key, title, owner, latest progress, and last update time. Selecting a card opens a simple detail pane with plain text, source links, and the existing core actions. Board counts are factual. Start with explicit buttons for valid transitions; drag-and-drop is optional polish and may never bypass claim, dependency, or QA rules.

Later add cross-project filtering, a table view if useful, quiet-session filters, activity/problem queues, and rich Markdown/document panes. All views call the same ticket operations as the CLI; closing the web process has no effect on agent coordination.

A detail panel shows type, rendered Markdown scope and criteria, optional checklist, parent/children, prerequisites/dependents, current claim, progress/decision history, and linked documents/artifacts. Filter the work table by ticket type as well as state. Keep an activity feed for “what changed since I last looked.” Display factual counts and timestamps instead of inferred completion percentages.

Add a Problems view with project, date, agent, ticket, and unreviewed-change filters. Show what failed, the workaround, later corrections, retrospective findings, and linked improvement tickets. Preserve access to the original reports underneath grouped patterns. Ticket details link to their reported workflow problems, including resolved incidents. Keep operational recovery, retrospective review, improvement delivery, and evidence of effectiveness visibly distinct.

The QA queue shows what changed, how to test it, validation evidence, and artifact/preview/PR links. Human actions are create/edit, prioritize, assign, block/unblock, accept QA, request changes, cancel/reopen, and explicit takeover. Assignment is recorded intent; the UI makes clear that no agent has started until a session claims the work.

Links can refer to PRs, commits, external previews, or external artifacts. For durable local evidence, copy explicitly attached files into managed storage; deleting a worktree should not delete its screenshots. The Markdown/document sidebar described below supports both those managed copies and explicitly registered live source files. The browser requests artifacts by ID; it cannot request arbitrary disk paths. Treat active HTML artifacts as downloads in the first release.

Bind the web server to loopback only, validate Host/Origin, and use a local session token with CSRF protection for mutations. Keep the browser's review action distinct from agent commands. Because everything runs under one OS account, this is an accidental-misuse boundary, not a security barrier against that user's agents. Remote browser access uses SSH forwarding; no public network listener, team accounts, or role-management system in version one.

## Linked Markdown documents and artifact sidebar

A ticket can link several documents with labels such as Design, Implementation plan, QA guide, or Reference. One artifact may be linked to an epic and multiple children without duplicating its content. Store these relationships explicitly so agents and the portal can discover them even when a document is also mentioned inline in the ticket body.

Clicking a linked Markdown document opens a resizable right sidebar while leaving the ticket visible. Render headings, paragraphs, tables, lists/task lists, fenced code, links, and images with a bundled Markdown renderer. Include the document title, source location, live/snapshot badge, version metadata, heading navigation, raw-source toggle, and download action. Preserve ticket scroll position; provide back navigation within the document pane and a full-width option. On narrow displays the pane can cover the ticket with a clear return action. Editing source documents in the portal is deferred; ticket Markdown editing uses a plain-text field with preview.

Support two explicit source modes:

- **Live reference:** resolve a registered worktree or document-root ID plus a relative path on the server. Read current bytes on open/refresh and display their hash plus observed commit and capture time; a worktree commit is context, not proof that uncommitted document bytes belong to that commit. Default plan/design links to this mode so agent edits are visible. If the source changes while open, show an update indicator and let the user refresh without losing their reading position. Never silently switch to another worktree or branch when a source goes missing.
- **Snapshot:** explicitly copy the document into managed artifacts with its source metadata and hash. Use this for accepted designs, QA evidence, or documents that should survive worktree deletion. A live reference may have a separately linked snapshot; the UI must clearly identify which version is shown. New captures create new artifact versions rather than overwriting old evidence.

Suggested CLI additions:

```text
conductor ticket create --type bug --body-file bug.md --json
conductor artifact link APP-42 --role plan --path docs/implementation.md
conductor artifact attach APP-42 --role design --path docs/design.md
conductor artifact read ART-12
```

`link` creates a live reference and records the current worktree identity. `attach` creates a managed snapshot. Explicitly register outside-repository document roots when needed. Resolve relative Markdown links and image paths against the displayed document's source directory, including heading anchors. Links to allowed Markdown files open in the same sidebar. Inline ticket-body links resolve only against an explicit associated worktree/document base; ambiguous paths produce an attach/select-source action instead of guessing a branch. External HTTP(S) links open normally; the first release does not fetch arbitrary remote pages for embedded rendering. A remote Markdown file can be explicitly imported for preview.

Enforce canonical root containment, including traversal, Windows path/UNC forms, symlinks and junctions, at file access time. Limit live previews to Markdown and supported passive image formats within explicitly registered roots; arbitrary source files require explicit attachment. Avoid check-then-open path races with safe handle-based resolution or an equivalent platform-specific containment check. A document link cannot grant access to another root. Missing or denied sources render an explanatory state with the original reference intact.

Disable raw HTML in Markdown and sanitize rendered output and link schemes. Serve local images and permitted document assets through authenticated same-origin routes. External images require an explicit load action; render no executable scripts, iframes, or active SVG content. Set preview size limits with a download fallback. These are renderer requirements, not extra agent workflow steps.

A snapshot initially captures the selected file only. Record and visibly flag relative assets/documents that were not captured; do not imply a standalone snapshot includes its entire linked tree. Agents can attach companion assets explicitly. All preview routes work identically on Windows and through the Linux VM's forwarded web port, without `file://` browser links or a network filesystem mount.

## Reliability and release acceptance

Schema migration runs exclusively before serving writes. Older binaries refuse unsupported newer schemas. Create an SQLite-consistent backup before destructive migrations; `conductor backup` uses the backup API rather than copying a live main database file. Document restore with all Conductor clients stopped. Include JSON export for portability, without making exports a second live source of truth.

Artifact imports copy into a temporary managed file and atomically publish it before recording the reference. A failed database commit may leave an orphan that maintenance can remove; it must not leave a successful attachment pointing to an incomplete copy.

Required validation is behavioral and uses real subprocesses and a real SQLite file:

1. Multiple agents racing for one ticket produce exactly one claim; independent ticket updates succeed.
2. A stale revision cannot overwrite newer content, and a revoked owner cannot submit work.
3. Dependency insertion races cannot create a cycle; blocked and QA-pending prerequisites do not unlock work.
4. Process termination cannot commit half a state change; a lost response followed by retry creates no duplicate.
5. Separate Git worktrees resolve to one project; separate unlinked clones remain distinct.
6. Backup/restore preserves tickets, claims, events, dependencies, workflow problems/reviews, and included managed evidence.
7. A browser can observe agent progress, request QA changes, then accept a later submission; stale browser edits conflict correctly.
8. Fresh Codex and Claude Code sessions discover a registered project through the user-level bootstrap, explicitly identify sessions, and complete work without hooks or repository instruction edits. Unregistered projects remain unaffected. Only if a hook is subsequently justified, test its lifecycle and continuation guard on the actual supported host version.
9. Concurrent problem appends preserve every observation; retries do not duplicate an occurrence; corrections preserve the original report.
10. A retrospective export is consistent and paginated; new reports and follow-ups after its watermark remain visible as unreviewed changes. Several reports can link to one improvement ticket without losing their evidence.
11. Ticket Markdown and a linked plan render side by side, including tables, code, anchors, and supported images; document task lists do not change structured ticket checklist state. Epic child navigation preserves context.
12. Live documents from two worktrees retain the correct source; refresh detects content changes, snapshots remain stable, and missing sources are explicit. Relative-link traversal, symlink/junction escapes, unsafe HTML/URLs, and oversized previews are rejected or safely handled.
13. Windows and Linux release builds pass storage/worktree tests. A Mac browser can load the Linux VM portal, render VM-resident Markdown/images, download artifacts, and perform QA through an actual SSH port forward; disconnect/reconnect does not interrupt agent writes. Test a different local forwarded port with explicit origin configuration.
14. When the optional worker skill ships, overlapping workers obey assignment/eligibility and claim exclusivity, wait when empty, pick up newly eligible work, skip blocked/review tickets, preserve scope across resume, and stop without silently abandoning ownership.
15. When the optional orchestrator ships, a single scoped supervisor can coordinate dependent work, reuse/start workers within its cap, preserve decisions in tickets, surface human QA, recover ambiguous launches without duplicate workers, and stop/resume while accurately reporting remaining children and claims. Unsupported host delegation has an explicit no-spawn fallback.

## Delivery slices

Deliver working increments instead of implementing the full target data model up front. See [Agent integration deliverables](2026-09-12-conductor-agent-deliverables.md) and [Portal UI design](2026-09-12-conductor-ui-design.md) for the corresponding phased contracts.

1. **CLI tracer:** a Windows executable with shared storage/access diagnostics, project/worktree discovery, stable agent identities and assignment, observed session availability, ticket create/show/list, explicit sessions, atomic claim/resume, progress append, submit/reject/accept, owner and human recovery release, and factual problem append/list/correction. Add request replay, the minimal work skill, and user-level bootstrap. Acceptance: real host access, assignment enforcement, concurrent claims, interrupted-response recovery, reject/rework/accept, release/reclaim with stale-owner rejection, and persisted progress/problems after restart. Skills default to isolated worktrees and record integration checks against the exact combined commit; PR links are ordinary notes initially. Exact commands and transitions are in the tracer specification. No web server, hooks, Markdown parser, or retrospective engine is required.
2. **Coordination increments:** add dependencies/basic parent context and address observed usability failures first. Optional checklists, richer context, backup/export, simple retrospective processing, and structured delivery/PR grouping are independently shippable enhancements, not one mandatory gate before the board. The retrospective can initially use exported problems, ordinary improvement tickets, and a Markdown report. PR grouping can initially use manually recorded links, without provider synchronization. Validate Linux CLI portability as a separate acceptance milestone.
3. **First portal:** a small Kanban board over the existing operations, card details with plain text, basic project filtering, observed owner/progress/blockers, and manual-QA actions. No second rules implementation and no Markdown/document viewer yet. Acceptance: watch live CLI updates appear, complete QA in the browser, close the portal and continue working, and access the Linux VM instance over an SSH tunnel from a Mac.
4. **Markdown and documents:** render ticket bodies, add registered document/artifact APIs and live links/snapshots, then a Markdown reading sidebar. Acceptance: read the correct worktree's plan beside a ticket, refresh changed content, and preserve captured evidence after deleting a worktree. Build the simplest read-only viewer first, then source toggles, navigation, and responsive/resize refinements.
5. **Refinements from use:** add Problems/activity views, structured retrospective coverage, richer filters and artifact handling as demonstrated needs arise. Improve installer diagnostics and docs continuously. Hooks remain conditional on evidence that they are necessary; do not reserve a mandatory hook phase.

Skills, registration, and instructions are versioned with the implemented commands. Ship only supported commands and teach the currently available subset at each increment. Do not wait for the final portal, advanced reports, or complete artifact system before installing and using the tracer.

The default worker skill executes one assigned ticket in a fresh host context and session, after prepared-work eligibility, claim recovery, and block/release work. Persistent polling is opt-in and additionally requires atomic filtered selection. It can be used before the portal but is not a prerequisite for shipping the board. The architecture review's recovery and acquisition gaps must be resolved before unattended work is enabled.

The optional orchestrator follows reliable single-job execution and prepared-work gates, reconciles required roles and approval state on each start, and validates one project's supervision through existing host agent tools. It does not delay the tracer or board. Higher-level goals remain epics; cross-provider runtime control is not assumed.

Keep the first usable release free of a built-in central agent scheduler or process supervisor, model routing, spending controls, Git/worktree management, automatic merge/deployment, remote sync, transcript ingestion, vector search, customizable workflows, and external tracker synchronization. User-started worker and orchestrator skills are within the later optional scope. The orchestrator may use existing host delegation/worktree capabilities, but durable scheduling/restarting of agent processes is outside the core. Other extensions should follow demonstrated usage needs.

## Source notes

The architecture, schemas, commands, and policies above are proposed product decisions. The following primary documentation supports the platform assumptions, checked on 2026-09-12:

- [SQLite WAL](https://sqlite.org/wal.html): concurrent readers with a serialized writer; shared local storage constraints.
- [SQLite transactions](https://sqlite.org/lang_transaction.html): write-transaction behavior and BEGIN IMMEDIATE.
- [Git rev-parse](https://git-scm.com/docs/git-rev-parse): shared Git directory discovery.
- [Go SQLite driver](https://pkg.go.dev/modernc.org/sqlite): a CGo-free SQLite driver candidate; pin and validate a supported release during implementation.
- [Codex hooks](https://learn.chatgpt.com/docs/hooks): lifecycle events and stop continuation behavior.
- [Codex skills](https://learn.chatgpt.com/docs/build-skills): supported skill discovery and distribution.
- [Claude Code hooks](https://code.claude.com/docs/en/hooks): lifecycle integration and continuation-loop guards.
- [OpenSSH](https://man.openbsd.org/ssh): local port forwarding for browser access to a Linux VM's loopback web server.

This file describes the target product. The CLI tracer, installer, work skill, and tracer integration are implemented; the team workflow and later portal increments remain separate deliverables. See the work board and tracer acceptance evidence for verified implementation status.
