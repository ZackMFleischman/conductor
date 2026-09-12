# Conductor CLI tracer specification

Implementation scope · 2026-09-12

This specification narrows the first increment of the main design and resolves its architecture-review contract gaps. Runtime verification remains part of implementation; this document does not claim that host access or software has been tested.

## Outcome and exclusions

Install a Windows CLI and a small work skill; use them from fresh Codex and Claude Code sessions in different worktrees to create, claim, update, submit, reject, reclaim, accept, and recover tickets. Append factual workflow problems and retain everything across restarts.

No portal, React build, Markdown parser, document storage, dependencies, epics, optional checklists, PR/delivery tables or hosting-provider integration, worker/orchestrator loops, retrospective engine, hooks, or MCP server in this increment. Markdown is stored as text. Git and Go are development prerequisites; users of the packaged CLI need Git for worktree discovery, not Go. Record implementation commits, optional PR links, and integration evidence in ordinary ticket notes in this increment; the main design defines their later structured representation.

Development baseline: Go 1.27.1 and modernc.org/sqlite v1.58.0, pinned with go.mod/go.sum. This is a specific starting baseline, not a claim that the driver is the latest release. Validate compatibility during Task 1. Windows amd64 is the first runtime acceptance target; keep portable code and cross-compile Linux, while Linux runtime support is verified in its following increment.

## Resolve the four review gaps

1. **Abandoned work:** owner release and an explicit human recovery release are tracer commands. Both invalidate ownership atomically and return work to ready. Recovery requires a reason and expected ticket revision. It does not stop the old process.
2. **Sandbox access:** setup is not successful until the selected agent hosts can invoke the CLI and write the shared database directory. The first task produces an actual SQLite/sidecar access probe. Registry failure means registration is unknown, not unregistered. Never create a fallback database silently.
3. **Claim recovery:** remove the separate credential cache. Store sessions and claims in SQLite. A claim's opaque random ID is an ownership fencing identifier, not a secret authentication capability. Owner operations require session ID, claim ID, and expected revision. Claim IDs are never reused; a released or submitted claim remains inactive. Recover the existing claim through session resume instead of generating a second claim or recovering a token from another file. This is cooperative same-OS-user coordination, consistent with the main design's trust boundary.
4. **Transitions:** creation defaults to ready. The following table is authoritative for the tracer.

| Operation | Transition | Caller and preconditions | Claim effect |
| --- | --- | --- | --- |
| create | absent → ready | Registered project; title and description | None |
| claim | ready → in_progress | Non-stopped session in same project; assignment compatible; expected revision; no active claim | New unique claim ID; one active claim per ticket and per session |
| note | state unchanged | Current session and claim; in_progress; append-only body | Retain |
| submit | in_progress → review | Current session/claim; expected revision; summary, evidence, QA steps | Release atomically |
| reject | review → ready | Explicit human action; expected revision; reason | None |
| accept | review → done | Explicit human action; expected revision | None |
| release | in_progress → ready | Current owner, or explicit human recovery; expected revision; reason | Invalidate atomically |

Manual QA is always required in this tracer; the configurable/no-QA branch follows later. The CLI exposes separate human actions using --human; skills never issue them on behalf of themselves. This is a workflow distinction, not an authentication boundary against the same OS user. All successful ticket mutations increment revision and append an event in the same transaction. Problem append does not mutate a ticket revision or require its ownership.

All database mutation requests carry a caller-known request ID. Persist request ID, operation/project/actor scope, payload hash, and result in the same transaction as the mutation. Replay the original result for an identical request; reject reuse with a different payload. A claim retry must also report whether the historical returned claim remains active. A lost claim response is recovered either by retrying the request or by resuming the same session. No request journal, claim secrets, or cache files are needed outside SQLite. Filesystem setup uses owned-file hashes and idempotent repair instead of the database request journal.

Expected revision is checked for state changes; appended notes compose without expected-revision checks but do require active ownership. A failed validation or conflict changes neither state nor business history. Retry SQLite contention with a bounded total budget; do not retry semantic conflicts automatically.

## Identity, assignment, and isolation

Register agents explicitly with a project-scoped unique name and stable UUID; role and provider are optional metadata. A role such as frontend is not an identity. Session start selects an existing identity and returns a new session UUID. Multiple sessions may belong to one identity; any may claim its assigned tickets, but each session has at most one active claim. Renaming and deleting identities are outside the tracer commands; their eventual implementation must preserve historic UUID references.

Tickets store nullable assigned_agent_id. Ticket creation can assign work; assignment changes require an expected revision and are rejected while a claim is active. An unassigned ticket is claimable by any registered identity; an assigned one only by sessions bound to that identity. A list filter is advisory: the mutation checks assignment inside the claim transaction. Assignment never launches an agent or grants a claim. Agent names are coordination labels, not authentication.

Derive busy from active claims. Separately store session last_seen_at and declared idle/stopped state; display registered-without-session and stale/unknown explicitly. Contact is updated only by explicit session commands or successful session-authored mutations; read-only dashboard queries do not create agent activity. A stopped session cannot claim or mutate as an owner. Stop fails with ACTIVE_CLAIM until the owner checkpoints and releases; stopping never implicitly releases ownership. Resume reconciles an existing non-stopped session and its claim. Quiet sessions are never automatically reclaimed. Availability lists are observations; the orchestrator must reconcile them with supported host status before dispatch.

Session start and claim observe the actual Git worktree root, branch (nullable for detached HEAD), and HEAD SHA before opening the transaction. Claims from a different project are rejected. Update session location on claim and record it on the claim event. The work skill normally creates or selects an isolated worktree through existing Git/host tools. The CLI returns a shared-checkout warning when claiming from the primary checkout, but permits intentional integration, documentation, and serial work there. This is a warning policy, not filesystem enforcement; no core Git manager or per-ticket worktree requirement.

Implementation tickets may be accepted against tested worktree commits. An ordinary integration ticket checks the combined result and records included source commits, target merged SHA, commands, and outcomes. It must not depend on implementation tickets whose own criteria already require that integration. A green result remains true for its recorded SHA and is insufficient for a newer target. In the tracer this comparison is performed by the skill before submission/acceptance; there is no background branch monitor. Direct integration and PR-based integration use the same evidence requirement; a PR may contain several tickets and is not a ticket state.

## Command contract

Every command accepts --json. stdout contains one JSON envelope; diagnostics go to stderr. Success: {"ok":true,"data":{...}}. Error: {"ok":false,"error":{"code":"REVISION_CONFLICT","message":"...","details":{...}}}. Exit codes: 0 success (including an unregistered context result), 2 usage, 3 conflict, 4 not found, 5 storage/access failure.

Global options: --home overrides CONDUCTOR_HOME/default application data; --project selects a project outside a checkout. --home is useful for tests but is never chosen implicitly based on CWD.

| Command | Required inputs / result |
| --- | --- |
| doctor --probe-write | Check database directory access with a temporary probe SQLite file, WAL and credential-free session persistence; return individual check outcomes |
| init --prefix APP --request KEY | Register canonical Git common directory; create store only here; return project_id |
| context --if-registered [--session S] | Read-only; return registered boolean and project; optionally current session/claim/ticket and bounded recent updates |
| agent register --name NAME [--role ROLE] [--provider PROVIDER] --request KEY | Return stable agent_id; duplicate name with another request returns AGENT_EXISTS |
| agent list | Return identities, sessions, claims, declared state and last_seen_at; never assert process liveness |
| session start --agent NAME_OR_ID --request KEY | Bind existing identity; return unique session_id, project_id, observed worktree/branch/commit |
| session resume --session S --request KEY | Update contact; return active claim or null; never start a new session or revive a stopped one |
| session idle --session S --request KEY | Declare idle only without active claim; update contact |
| session stop --session S --request KEY | Mark stopped only without active claim; preserve history |
| ticket create --title TEXT --body-file PATH [--assigned-to NAME_OR_ID] --request KEY | Return ticket with id, project_id, title, body, state, revision, assigned_agent_id |
| ticket assign ID --to NAME_OR_ID_OR_none --expect-revision N --request KEY | Resolve in this project; reject active claim; increment revision and append event |
| ticket list [--state STATE] [--assigned-to NAME_OR_ID_OR_none] | Return project tickets, bounded limit/cursor; none means unassigned |
| ticket show ID | Return ticket and latest 20 events |
| ticket claim ID --session S --expect-revision N --request KEY | Return ticket and claim_id; an already active same-session claim is discoverable through resume |
| ticket note ID --session S --claim C --body-file PATH --request KEY | Append progress and return new revision/event ID |
| ticket submit ID --session S --claim C --expect-revision N --summary-file P --evidence-file P --qa-file P --request KEY | Persist all three bodies; release claim; return review ticket |
| ticket accept ID --human --expect-revision N --request KEY | Return done ticket |
| ticket reject ID --human --expect-revision N --reason TEXT --request KEY | Return ready ticket |
| ticket release ID --session S --claim C --expect-revision N --reason TEXT --request KEY | Owner checkpoint/release; return ready ticket |
| ticket release ID --human --expect-revision N --reason TEXT --request KEY | Recovery release; return ready ticket |
| problem add [--ticket ID] --session S --body-file PATH --request KEY | JSON body: summary, expected, actual; optional correction and evidence text; return problem ID |
| problem append ID --session S --body-file PATH --request KEY | Append a correction/evidence note; preserve initial report |
| problem list | Return bounded project reports with appended notes |
| setup --agents codex,claude [--apply] | Preview by default; apply owned user-level skill/bootstrap files only when requested |
| setup --remove --agents codex,claude [--apply] | Preview/remove Conductor-owned integration only; never remove ticket data |

Use UUID strings for internal IDs and project-prefix numeric display keys. Allocate display keys within a transaction. Request IDs are client-provided opaque strings, bounded in length; the skill uses a session/operation sequence and repeats that ID on retry.

## Persistence and boundaries

Only projects, worktrees, agents, sessions, tickets, claims, events, problems, and requests are required. Store schema version using SQLite user_version. Problem follow-ups use append-only events. No generic entity engine or prospective orchestration tables.

Use one SQLite connection per short-lived CLI process, WAL, foreign_keys=ON, busy_timeout=5000, and synchronous=FULL. Use BEGIN IMMEDIATE on the same connection for read-validate-write operations; statements use bound parameters. Session/claim uniqueness has database indexes. No shell/network/file reads inside database transactions. Read body files before mutation begins and hash the canonical parsed request payload.

Filesystem writes are limited to the data directory, explicit setup destinations, and probe/test fixtures. Actual source files are never edited by the tracer. The probe closes all handles and removes only files it created, using literal paths and verified containment.

## Host access checkpoint

At initial implementation, inspect actual installed host versions and effective settings. The documented candidates are Codex sandbox_workspace_write.writable_roots and Claude permissions.additionalDirectories; existing managed policy may limit them. Merge only the exact application data directory, preserve existing configuration, and verify in fresh sessions. Do not claim a config edit proves effective OS access, disable sandboxing, or approve all shell commands. If the current managed host cannot grant access, record the limitation before expanding implementation scope; do not silently build a service.

Discovery returns registered=false only after reading a healthy registry and finding no match, or when no store exists yet. An existing but unreadable/corrupt/unsupported database returns REGISTRY_UNAVAILABLE with registration unknown. --if-registered never creates or migrates storage.

## Done evidence

The first tracer is usable when both fresh agent hosts access the shared store from different worktrees, one of two claim attempts wins, wrong-assignee claims fail, progress/problem reports survive restarts, claim recovery works after a lost response, and the complete reject/rework/accept loop works. Human recovery release must invalidate a stale owner. Agent lists must distinguish claims from old contact/idle reports. Integrate two worktree changes and record validation against the exact combined commit. An unrelated repository stays unregistered. Setup round-trip preserves existing instructions and hooks. Do not call this complete based solely on unit tests or cross-compilation.

Sources checked for this plan: [Go downloads](https://go.dev/dl/), [SQLite driver](https://pkg.go.dev/modernc.org/sqlite@v1.58.0), [Codex configuration](https://learn.chatgpt.com/docs/config-file/config-reference), [Claude Code permissions](https://code.claude.com/docs/en/permissions).
