# Foundation ticket commands

These commands require project registration, not planning policy or a managed team. Prefix them with the installed `conductor` executable and its configured `--home PATH`; append `--json`. Start a live session under a stable agent identity, retain request keys for retries, and refresh revisions before changing state.

```text
ticket create --title "Fix the import error" --body-file description.md --session S --request create-1
ticket edit APP-1 --session S --expect-revision N --body-file specification.json --request edit-1
ticket claim APP-1 --session S --expect-revision N --request claim-1
ticket note APP-1 --session S --claim C --body-file progress.md --request note-1
ticket block APP-1 --session S --claim C --expect-revision N --reason "Missing fixture; tracked in APP-2" --request block-1
ticket unblock APP-1 --session S --expect-revision N --body-file unblock.json --request unblock-1
```

`specification.json` replaces title, description, parent and dependencies; include the whole intended specification:

```json
{"title":"Fix the import error","body":"Purpose, scope, acceptance criteria and Markdown document links.","kind":"bug","parent_id":"APP-3","dependencies":["APP-2"],"reason":"Record scope and prerequisite"}
```

Kind is a project convention, such as bug, feature, epic or integration. Parentage groups tickets; dependencies order execution. Parent/dependency cycles and cross-project references are rejected. Omitted parent/dependencies clear them. Release a plain ticket's claim before editing its specification. `unblock.json` contains `{"reason":"Fixture delivered in commit ..."}`. Unblocking returns a plain ticket to ready; unfinished dependencies still prevent acquisition.

```text
ticket submit APP-1 --session WORKER --claim C --expect-revision N --commit SHA --summary-file summary.md --evidence-file evidence.md --qa-file validation-steps.md --request submit-1
ticket accept APP-1 --session REVIEWER --expect-revision N --validation-file decision.json --request accept-1
ticket reject APP-1 --session REVIEWER --expect-revision N --validation-file decision.json --reason "Observed failure and reproduction" --request reject-1
```

`decision.json` records provenance and evidence:

```json
{"commit":"SHA","criteria":"Criteria inspected","evidence":"Commands, observed results, and review notes","context_id":"actual-host-context","checks":{"unit":"pass"}}
```

A rejection may contain failed checks. A plain ticket records the live actor and supplied evidence; it does not claim independent review merely because another session recorded the decision. Existing legacy tickets retain human decisions after migration. Use `--human` only for an explicitly authorized human action. Optional policy may require independent identity, an exact submitted commit, specific checks, or human involvement; see [workflow commands](workflow-commands.md).

`ticket show APP-1` exposes universal metadata and bounded history, plus optional workflow data when present. `ticket list --state STATE --assigned-to AGENT` filters the board. A visible ready state is not a promise that dependency, policy or managed-session checks will allow acquisition. Claim checks and the state change share one transaction.

Problem reporting remains independent of ticket claims and teams. Use the work skill's existing `problem add` / `problem append` commands. [Retrospective processing](retrospective-commands.md) adds optional grouping, remedies and deferral; [team coordination](team-commands.md) adds managed agent lifecycle restrictions.
