# Workflow commands (contract v2)

All commands below run after the installed executable, optional `--home PATH`, and project selection. Add `--json`. Commands write transactionally with a unique `--request KEY`; identical replay returns the original result. Body files are UTF-8 JSON, bounded to 256 KiB. Ticket IDs accept UUIDs or display keys. Refresh `ticket show ID` before every revision-checked mutation.

Projects opt in explicitly. Existing tickets keep legacy human acceptance; opting in does not retroactively change them. New tickets inherit policy snapshots and start in `draft`. Changing project defaults does not rewrite inherited ticket policies. Only explicit local human configuration can change core policy; a worker session cannot weaken its gates.

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

`--human` records an explicitly authorized human action. Agents must not use it to impersonate human QA or bypass a retained gate. Delegated ticket preparation and authorization use the active team coordinator session and coordination token.

## Drafts, specifications, critiques, preparation, authorization

```text
ticket create --title "Implement bounded scope" --body-file criteria.md --session S --request create-1
ticket edit ID --session COORDINATOR --coordination TOKEN --expect-revision N --body-file spec.json --request edit-1
ticket critique ID --session REVIEWER --expect-revision N --body-file critique.json --request critique-1
ticket dispose ID --session COORDINATOR --coordination TOKEN --expect-revision N --body-file disposition.json --request disposition-1
ticket prepare ID --session COORDINATOR --coordination TOKEN --expect-revision N --body-file prepared.json --request prepare-1
ticket authorize ID --session COORDINATOR --coordination TOKEN --expect-revision N --body-file grant.json --request grant-1
```

Human-controlled ticket operations use `--human` in place of `--session` and `--coordination`. Critiques always use a live reviewer session. Agent coordinator mutations require both the live session and current exclusive coordination token, including after recovery.

`spec.json` is a full replacement specification; omitted parent/dependencies clear them:

```json
{"title":"Implement bounded scope","body":"Purpose, scope, criteria, context links, interfaces, conflicting files and validation mapping.","parent_id":"WF-1","kind":"implementation","dependencies":["WF-2"],"reason":"Material change summary"}
```

Optional `validation_mode` and `required_checks` replace inherited validation only with human authority. Kind describes the work (`implementation`, `planning`, `review`, `integration`, `improvement`, or another project convention); it does not bypass gates. Parentage groups scope, dependencies order work; both reject cycles and cross-project references. Requirement edits increment `spec_revision`, invalidate preparation/authorization on the edited ticket and its transitive dependents, and bump their observable ticket revisions. Active ownership remains intact, with `paused=true`; notes/checkpoints and release remain possible, but submission is blocked until reprepared and authorized. Ordinary progress notes do not change the specification revision.

Critique body: `{"context_id":"fresh-host-review-id","body":"Findings and evidence, or no findings","significant":true}`. A stable identity that implemented the ticket or authored its version cannot provide independent critique. The context ID records the host context supplied by the supervisor; Conductor does not launch or inspect host conversations. Preparation records immutable current-version critique inputs. Significant findings require a disposition: `{"critique_id":"UUID","body":"Resolution or explicitly authorized acceptance, with reason"}`.

Prepared body: `{"body":"Exact plan revision/digest, selected scope, dependency snapshot, criterion-to-method/role/stage/evidence mapping, delivery preference and resource limits"}`. This is attributable preparation evidence. Independent planning policy requires a critique of the current specification; unresolved significant findings block preparation. Use at most three substantive review rounds by default and escalate remaining significant findings to the retained decision maker.

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

`block` atomically records the blocker and releases the claim; the worker can then stop its session. `unblock.json` is `{"reason":"Blocker resolved with evidence ..."}`. Normal `ticket release` remains available; stale owners cannot note, submit, block or release. Every claim checks assignment, active claims, live session, preparation, authorization, pause/block state, completed dependencies, and active retrospective deferrals in the same transaction. Claim is exact-ticket; this version has no filtered claim-next command.

Managed submissions require a tested commit. Legacy submissions retain the prior CLI. `validation.json`:

```json
{"commit":"TESTED_COMMIT","criteria":"Exactly completed acceptance criteria","evidence":"Review evidence, commands, results and provenance","context_id":"fresh-validator-host-id","checks":{"unit":"pass","integration":"pass"}}
```

Validation must match the submitted commit and current specification. Every named required check must equal `pass`. Independent mode requires a live validator session, fresh host context reference, and a stable agent identity with no implementation claim in any attempt of the ticket. A second session under the implementation identity is rejected. Automated mode records named checks and provenance and may be recorded by the implementing session; this is visibly automated acceptance, never represented as independent review. Human mode uses `ticket accept ID --human --validation-file validation.json ...`; legacy tickets still require `--human` and do not require the new JSON evidence file.

`ticket show` includes specification/policy metadata and bounded histories of versions, critiques/dispositions, validations and dependencies (100 each with explicit truncation). Existing notes/events remain available. `ticket list --state draft|blocked|ready|in_progress|review|done` lists the corresponding state; a displayed `ready` state is not a promise that dependencies or deferral permit acquisition.

## Transaction helper contracts

`CreateWorkflowDraftTx(ctx, conn, project, session, title, body, parentID, kind) (TicketRecord,error)` creates a draft and immutable version inside the caller's mutation transaction; workflow opt-in is required. The caller journals its own surrounding event. `CheckWorkflowEligibilityTx(ctx, conn, project, ticketID) error` checks inherited gates, dependencies and retrospective deferral in an existing acquisition transaction. Team worker launch uses the same helper after integration. Neither helper commits independently.
