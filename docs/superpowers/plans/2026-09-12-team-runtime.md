# Team workflow implementation plan

**Goal:** Ship the reviewed workflow prerequisites and install usable skills before starting the separately approved case study.

**Spec:** [Team workflow](../specs/2026-09-12-conductor-team-workflow.md).

**Architecture:** Extend the existing Go/SQLite core and CLI. The host executes agents; Conductor records plans, coordination and evidence. Keep mutations transactional and request-idempotent. No service, hooks, portal, or sample-project implementation in this increment.

## Execution and file ownership

| Task | Deliverable and owned files | Dependencies / verification |
| --- | --- | --- |
| R0 | Root: version-2 migration loader and migration tests in internal/store; preserve read-only discovery and legacy data. | First. Test legacy migration, refusal of newer versions, rollback and read-only access. |
| R1 | Ticket worker: internal/core/tickets.go, new preparation/validation files, internal/cli/tickets.go and workflow command, migrations/10-workflow.sql, focused tests. | R0. Draft/readiness, version approval, dependency/parent cycles, block/release, independent identity validation, stale approval/owner rejection. |
| R2 | Team worker: internal/core/team.go, internal/cli/team.go, migrations/20-team.sql, focused tests. | R0. Exclusive coordinator, immutable scope/profile, persisted launches, child acknowledgements, capacity, unknown launches, resume/stop/recovery, readiness and events. |
| R3 | Retrospective worker: internal/core/retrospective.go, internal/cli/retrospective.go, migrations/30-retrospective.sql, focused tests. | R0. Incremental reports including old-report appends, bounded coverage, retry/dedup decisions, linked draft remedies, deferral and recurrence. R1 integration for ticket metadata. |
| R4 | Root: all six skills, embedded asset packaging, setup upgrade tests and CLI version/capabilities. | R1-R3 commands fixed. Existing bootstrap opt-in preserved; test owned upgrades and fresh-context skill use. |
| R5 | Independent integration/review: subprocess tests and live smoke fixtures, final review and installation. | R1-R4. Full test/vet/build, real managed workflow smoke, review fixes and exact-source evidence. |

R1/R2/R3 use separate worktrees from R0. One implementer per ticket; a fresh reviewer checks each completed deliverable. Root owns integration, shared migration loader, packaging and skills. Shared-store dogfood uses the installed tracer until the candidate is verified, then an explicit consistent backup precedes migration. Never run an incomplete version-2 branch against the real registry.

## Common interfaces

- Retain Service.Mutate and Store.Write for every write, request payload replay and actor attribution. Use project-scoped lookups, revision checks, live sessions, and the existing event log.
- Version 2 executes embedded migrations/*.sql in filename order in one transaction. Each worker owns only its numbered SQL file; root owns the loader. Do not change version-1 schema.sql. Ticket table rebuilding must preserve all keys/data and foreign key references.
- Commands use existing Register/Parse/OpenProjectReadOnly/OpenProject and bounded JSON input files where multi-field payloads are clearer. Include --help and actual runnable examples in a feature command reference. No direct SQL in skills.
- R1 owns ticket metadata and policy. R3 creates linked remedies through an agreed transaction-level helper, or records links to already-created draft tickets if atomic creation cannot safely compose; final integration must prove no duplicate remedy under retry.
- R2 owns coordination/launch authorization; R1 uses the recorded coordinating session for managed policy mutations after integration. Local explicit human configuration is separate from worker activity. Workers cannot weaken their own gates.
- New team workflows opt in; legacy human-QA tickets remain intact. Preparation review, execution grant, result validation and delivery permissions are distinct. No self-validation by switching session IDs.
- Startup stores readiness observations, not inferred process liveness. Host state is supplied through supported native tools; no custom launcher. Fresh worker means fresh host context plus tracker session.

## Acceptance procedure

For each task, write behavior tests first, confirm the intended failure, implement, and run focused checks. Review command contracts before dependent skills reference them. Root reruns the integrated suite once combined, then fixes/rechecks concrete failures only. Independently exercise a small fixture (not the reading-list case study): prepare -> approve under delegated policy -> launch intent -> child acknowledgement -> claim -> submit -> distinct validator acceptance; report -> decision -> linked remedy -> recurrence; stop/resume and ambiguous launch recovery. Compare persisted records with actual host/session behavior.

Record source SHAs, test commands/results and limitations in docs/testing/team-workflow-acceptance.md. Update PROGRESS.md as a simple work board. Publish the finished branch to the existing Conductor remote. The sample project and scope still require explicit approval; a GitHub Project board is unnecessary, and a remote is optional unless PRs are in its scope.
