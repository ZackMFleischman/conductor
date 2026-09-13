# Workflow commands (schema v3)

All commands below run after the installed executable, optional `--home PATH`, and project selection. Add `--json`. Commands write transactionally with a unique `--request KEY`; identical replay returns the original result. Body files are UTF-8 JSON, bounded to 256 KiB. Ticket IDs accept UUIDs or display keys. Refresh `ticket show ID` before every revision-checked mutation.

Projects opt in explicitly. Ordinary new tickets work without planning policy; migrated legacy tickets keep human acceptance. Opting in does not retroactively change existing tickets. New tickets inherit policy snapshots and start in `draft`. Changing project defaults does not rewrite inherited ticket policies. Only explicit local human configuration can change core policy; a worker session cannot weaken its gates.

## Project policy and original intent

```text
workflow configure --human --body-file policy.json --request policy-1
workflow show
workflow intent
workflow amend --human --body-file amendment.json --request amendment-1
```

Initial `policy.json`:

```json
{
  "expected_revision": 0,
  "intent": "Original user purpose, authorized scope, non-goals and retained decisions.",
  "reason": "Explicit user authorization for this policy.",
  "execution_mode": "delegated",
  "plan_review": "independent_agent",
  "validation_mode": "independent_agent",
  "required_checks": ["unit", "integration"]
}
```

`execution_mode`: `human` or `delegated`. `plan_review`: `lightweight` or `independent_agent`. `validation_mode`: `human`, `independent_agent`, or `automated`. Automated mode requires at least one named check. Updates use the current policy `expected_revision` (or `--expect-revision N`) and a reason. Original intent is immutable; later configure calls do not overwrite it. `amendment.json` contains `{"intent":"Explicit subsequent user amendment"}`; amendments append attribution and history.

`--human` records an explicitly authorized human action. Agents must not use it to impersonate human QA or bypass a retained gate. Agents can edit, dispose findings, prepare, and unblock tickets with a live standalone session under the standing human configuration grant and inherited ticket policy. Execution authorization additionally requires delegated execution mode; human execution mode retains only that authorization for the user. If any non-stopped team exists, use its current coordinator session and token; stopping or released runs require reconciliation.

## Drafts, specifications, critiques, preparation, authorization

```text
ticket create --title "Implement bounded scope" --body-file criteria.md --session S --request create-1
ticket edit ID --session COORDINATOR --coordination TOKEN --expect-revision N --body-file spec.json --request edit-1
ticket critique ID --session REVIEWER --expect-revision N --body-file critique.json --request critique-1
ticket dispose ID --session COORDINATOR --coordination TOKEN --expect-revision N --body-file disposition.json --request disposition-1
ticket prepare ID --session COORDINATOR --coordination TOKEN --expect-revision N --body-file prepared.json --request prepare-1
ticket authorize ID --session COORDINATOR --coordination TOKEN --expect-revision N --body-file grant.json --request grant-1
```

For standalone planning, use `--session S` and omit `--coordination` in the examples above. When execution mode is human, use `--human` for authorization after agent preparation. Human-controlled ticket operations use `--human`. Critiques always use a live reviewer session. With a non-stopped team, planning mutations require its live coordinator and current exclusive token, including after recovery. They do not require worker dispatch readiness. Historical managed child sessions cannot become standalone policy actors.

`spec.json` is a full replacement specification; omitted parent/dependencies clear them:

```json
{"title":"Implement bounded scope","body":"Purpose, scope, criteria, context links, interfaces, conflicting files and validation mapping.","parent_id":"WF-1","kind":"implementation","dependencies":["WF-2"],"reason":"Material change summary"}
```

Optional `validation_mode` and `required_checks` replace inherited validation only with human authority. Kind describes the work (`implementation`, `planning`, `review`, `integration`, `improvement`, or another project convention); it does not bypass gates. Parentage groups scope, dependencies order work; both reject cycles and cross-project references. Requirement edits increment `spec_revision`, invalidate preparation/authorization on the edited ticket and its transitive dependents, and bump their observable ticket revisions. Active ownership remains intact, with `paused=true`; notes/checkpoints and release remain possible, but submission is blocked until reprepared and authorized. Ordinary progress notes do not change the specification revision.

Critique body: `{"context_id":"fresh-host-review-id","body":"Findings and evidence, or no findings","significant":true}`. A stable identity that implemented the ticket or authored its version cannot provide independent critique. The context ID records the host context supplied by the supervisor; Conductor does not launch or inspect host conversations. Preparation records immutable current-version critique inputs. A significant critique submitted after preparation invalidates preparation and authorization on the ticket and transitive dependents, pauses active work, and blocks further submission until disposition, preparation, and authorization. Significant findings require a disposition: `{"critique_id":"UUID","body":"Resolution or explicitly authorized acceptance, with reason"}`.

Prepared body: `{"body":"Exact plan revision/digest, selected scope, dependency snapshot, criterion-to-setup/preconditions/method/role/stage/expected-observation/exact-source/provenance mapping, delivery preference and resource limits"}`. This is attributable preparation evidence. Independent planning policy requires a critique of the current specification; unresolved significant findings block preparation. Use at most three substantive review rounds by default and escalate remaining significant findings to the retained decision maker.

Grant body: `{"reason":"Current prepared scope is covered by the standing user grant, reference ..."}`. Preparation and execution grant are separate. Human execution mode only accepts `--human` authorization. Policy snapshots stay visible on `ticket show`.

## Execution, blockers, result validation

```text
ticket claim ID --session WORKER --expect-revision N --request claim-1
ticket note ID --session WORKER --claim CLAIM --body-file checkpoint.md --request note-1
ticket block ID --session WORKER --claim CLAIM --expect-revision N --reason "Missing input and next action" --request block-1
ticket unblock ID --session COORDINATOR --coordination TOKEN --expect-revision N --body-file unblock.json --request unblock-1
ticket submit ID --session WORKER --claim CLAIM --expect-revision N --commit TESTED_COMMIT --summary-file summary.md --evidence-file evidence.md --qa-file validation-steps.md --request submit-1
ticket accept ID --session VALIDATOR --expect-revision N --validation-file validation.json --request accept-1
```

`block` atomically records the blocker and releases the claim; the worker can then stop its session. `unblock.json` is `{"reason":"Blocker resolved with evidence ..."}`. Normal `ticket release` remains available; stale owners cannot note, submit, block or release. Every claim checks assignment, active claims, live session and completed dependencies. Optional policy, active retrospective deferrals and managed-team restrictions add checks in the same transaction. Claim is exact-ticket; this version has no filtered claim-next command.

Policy-governed submissions require a tested commit. Legacy submissions retain the prior CLI. `validation.json`:

```json
{"commit":"TESTED_COMMIT","criteria":"Exactly completed acceptance criteria","evidence":"Review evidence, commands, results and provenance","context_id":"fresh-validator-host-id","checks":{"unit":"pass","integration":"pass"}}
```

Validation must match the submitted commit and current specification. Review each criterion against its planned evidence contract: required setup/preconditions, execution method, actual observation, exact checkout/commit, and executor/raw-artifact provenance. Verify that each cited artifact contains the exact fact attributed to it; split facts across citations when their sources differ. Missing or mismatched precondition evidence makes that criterion incomplete even when a command or named check passed; do not issue complete acceptance. Directory names and successful commands do not prove ownership or execution identity. Acceptance requires every named check to equal `pass`, but those structural checks do not establish the semantics of free-text criteria or evidence. Rejection uses `ticket reject ID --session VALIDATOR --expect-revision N --reason "Failed criterion" --validation-file validation.json --request reject-1`; failed or missing passing-check results are allowed. Rejection still requires structured attribution, matching submitted commit/current specification and the selected validation authority or independent identity. The rejected decision and evidence are recorded once, and the ticket returns to ready for rework. Independent mode requires a live validator session, fresh host context reference, and a stable agent identity with no implementation claim in any attempt of the ticket. A second session under the implementation identity is rejected. Automated mode records named checks and provenance and may be recorded by the implementing session; this is visibly automated acceptance, never represented as independent review. Human mode uses `ticket accept ID --human --validation-file validation.json ...`; legacy tickets still require `--human` and do not require the new JSON evidence file.

Validation history is append-only. Evidence produced after a decision may support a later prospective conclusion, but it does not repair or rewrite the evidence basis of the earlier decision. Record the original decision's incomplete criterion coverage and cite the later addendum separately.

`ticket show` includes specification/policy metadata and bounded histories of versions, critiques/dispositions, validations and dependencies (100 each with explicit truncation). Existing notes/events remain available. `ticket list --state draft|blocked|ready|in_progress|review|done` lists the corresponding state; a displayed `ready` state is not a promise that dependencies or deferral permit acquisition.

## Transaction helper contracts

`CreateWorkflowDraftTx(ctx, conn, project, session, title, body, parentID, kind) (TicketRecord,error)` creates a ticket and immutable version inside the caller's mutation transaction. Configured projects produce gated drafts; projects without policy produce ordinary ready tickets. The caller journals its own surrounding event. `CheckWorkflowEligibilityTx(ctx, conn, project, ticketID) error` checks inherited gates and retrospective deferral in an existing acquisition transaction. Foundation dependencies use `CheckTicketDependenciesTx`; acquisition and team worker launch compose these checks. Neither helper commits independently.
