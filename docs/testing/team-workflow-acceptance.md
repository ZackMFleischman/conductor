# Layered workflow acceptance

Observed on Windows on 2026-09-12 (America/Los_Angeles). This validates the internal workflow prerequisites. The proposed reading-list case study has not started and still requires the user's project/scope approval.

## Installed source and integration checks

- Source: `b1399e8a688daef0affde575284e74e6c75f2064`, clean build tree.
- Installed CLI: `0.2.0-dev`, schema `3`, skill contract `layers-v1`.
- Windows amd64 SHA-256: `1b41e13117aa1c28b3c0fa5e7a0c314ac5fbe9c2984f46cf846581347af7bb15`.
- Linux amd64 SHA-256: `d3c1c2838214df67077a7b7f00974c835360b707d1f9765af43dd28ed0819459`.
- Fresh `go test ./... -count=1`: CLI, core, gitctx, setup, store, and integration packages passed; no package was excluded. `go vet ./...` exited 0.
- The reviewed layer plan and implementation preserve ordinary tickets without team/planning requirements, optional policy checks inside claim transactions, managed identity/assignment restrictions, historical request replay, and existing human acceptance gates. Review corrections include historical submission provenance, production claim/rejection wiring, and allowing agent preparation while retaining human execution approval.
- The installed binary migrated the real existing registry to schema 3 while preserving the coordinator's active CON-4 claim. The consistent pre-migration backup is `conductor.db.pre-v3-1789260755508803400` in the configured data home. The former executable was also retained.
- All six owned skills were upgraded for Codex and Claude; a subsequent installer preview proposed zero changes. A fresh Claude context (`21d180ea-1512-4599-a5c2-9d4d6b048487`) read the new work/orchestration/planning/worker/improver instructions and correctly described optional orchestration and the `layers-v1` contract. That read-only check did not test Claude team orchestration; earlier standalone Claude coordination is recorded in [tracer acceptance](tracer-acceptance.md).

## Fresh Windows sandbox

With the user's explicit authorization, official Codex sandbox setup enabled elevated restricted accounts and the shared Conductor data root. Existing parent tasks retain their captured permissions; their old access failures do not describe a freshly started sandbox.

The fresh `command/exec` probe retained `workspaceWrite` restrictions. The installed binary returned registered=true for the worktree, all five doctor checks passed (open, WAL creation, persistence, reopen, cleanup), and Git status exited 0. The Windows canonical-path fix resolves repository handles without requiring ancestor-directory enumeration. Missing paths remain errors. Nonfatal access warnings for the private Git ignore file remain; this probe does not establish access to unrelated user files.

These results supersede the historical unelevated-sandbox limitation in [local host checks](live-host-checks.md). Linux was cross-built, not run.

## Native workflow fixture

The fixture deliberately seeds an obsolete `--data-dir` flag in a tiny PowerShell runbook. It is injected test data, not a production defect or the proposed sample application. It uses an explicitly selected isolated SQLite home, an initial repository, and a separate worker worktree under the ignored `.superpowers/native-smoke` directory. It never substitutes this home for an inaccessible production registry.

The standing fixture policy delegates execution, uses lightweight preparation for this one-line correction, and requires independent-agent result validation with a named `probe` check. The run permits four host slots including the coordinator and requires a live improver. Actual native spawn/status results are recorded separately from database sessions; readiness acknowledgements are made by the children themselves.

Project: `8ce2e2d7-a1be-41c6-a77f-0a69d1a25516` (SMK). Run: `477deade-9d02-4c44-b604-9b7fc6cc5e5e`. Baseline source: `2b9221a75a341fc000f15875e2c22983b97acb1c`.

| Stage | Observed result |
| --- | --- |
| Seeded failure | Baseline wrapper exited 2 with `USAGE: unknown command: --data-dir`; original evidence remained intact. |
| Improver | Fresh `/root/smoke_improver` created SMK-2 under the single Workflow Improvements epic. A second pre-delivery report reused the same remedy. |
| Preparation | Recorded specification 1 preparation and delegated authorization preceded the sole implementation claim. |
| Worker | Fresh `/root/smoke_worker` claimed its exact assignment from `smoke/fix`, changed only the stale flag, observed failure then success, committed `33befda62a3088facd550cf6ed5cc8487d3e07d5`, submitted, and stopped. All five doctor checks passed at the committed source. |
| Independent validation | Fresh `/root/smoke_validator`, a distinct stable identity with no implementation participation, inspected the diff and reran the exact submitted commit using another previously absent probe home. Its structured `independent_agent` decision accepted SMK-2 at revision 8. No human QA was impersonated. |
| Integration and later repeat | The fixture main branch fast-forwarded to the accepted SHA. Fresh `/root/smoke_repeat` ran the affected workflow after acceptance/integration, at `2026-09-13T01:21:02Z`, using a third clean probe home. Exit 0 and all five Boolean checks passed. This observer made no implementation claim or source change. |
| Effectiveness | The improver directly inspected the later repeat, processed its report append, and recorded `improved` with the explicit limit of one successful synthetic repeat. Three decisions cover original report seq 2, pre-delivery follow-up seq 16, and later repeat seq 46. Watermark 46, pending 0, no open batch, one remedy. |
| Shutdown | Run stopped at revision 33/epoch 11 with coordination released. All four native child hosts completed, all five tracker sessions stopped, and zero active claims remained. |

The final audit opened SQLite read-only in one read transaction and checked integrity/foreign keys, exact worker assignment and worktree, prepare-before-authorize-before-claim-before-submit-before-accept ordering, distinct worker/validator identities and hosts, matching submission/decision commit, complete report-event coverage, remedy deduplication, effectiveness, and shutdown. Every assertion passed. The structured record is [team workflow evidence](team-workflow-evidence.json); detailed local snapshots and probe transcripts remain under `.superpowers/native-smoke/evidence/`.

Two real workflow issues were recorded separately from the injected defect: **CON-P27** captures command-help/reference discovery corrections, including a corrected audit harness path; **CON-P28** captures the acknowledgement traffic required when a healthy team gains a short-lived child. These are improvement inputs, not silently repaired policy changes. The final review of the human-execution preparation correction found no remaining actionable issue.

## Scope of the evidence

This small run tests actual use of the installed skills and CLI, not unattended production reliability. Native host identity/status remain supervisor-supplied observations; Conductor does not launch or independently inspect processes. Synthetic repeated reports establish deduplication behavior, not the prevalence of a real workflow defect. A successful later repeat is bounded effectiveness evidence, not proof of permanent improvement. The internal coordinator also performed database audits; it was not the separate bystander planned for the case study. Native concurrent implementation, coordinator crash recovery, milestone release, and Linux execution were not exercised in this small run; deterministic integration tests cover the implemented recovery and deferral rules.

The React/MUI portal is a separate parallel effort. Its frontend fixtures and proposed live API contract are not part of this acceptance or a prerequisite for the case study.
