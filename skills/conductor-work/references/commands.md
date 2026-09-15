# Conductor work commands

For hierarchy, dependencies, blockers and ordinary agent decisions without planning or teams, read [foundation ticket commands](ticket-commands.md). Existing legacy tickets retain their human decision requirement. Optional workflow and managed-team checks add restrictions only where configured.

All commands return one JSON envelope: `{"ok":true,"data":...}` or `{"ok":false,"error":{"code":...,"message":...,"details":...}}`. Use `--json`. Exit codes: 0 success, 2 usage, 3 conflict, 4 missing entity, 5 storage/access failure. Read `data` only after checking `ok`.

Use the absolute executable/data paths recorded in the bootstrap. PowerShell uses `& 'C:\Program Files\Conductor\conductor.exe' --home 'C:\Users\you\AppData\Local\Conductor' ...`; Bash uses `'/absolute/path/conductor' --home '/absolute/data/path' ...`. Never derive the data home from the working directory. `--home` overrides `CONDUCTOR_HOME`, which overrides the platform default. `--project PROJECT_ID` selects a registered project outside a checkout.

`KEY` is a caller-retained unique request ID per logical mutation; repeat it unchanged only when retrying that same operation/payload. `S`, `C`, and `N` are the returned session UUID, active claim UUID, and current ticket revision. `ID` accepts the ticket identifier provided by the CLI.

| Purpose | Command after executable/global options |
| --- | --- |
| Discovery | `context --if-registered [--session S] --json` |
| Access probe | `doctor --probe-write --json` |
| Explicit registration | `init --prefix APP --request KEY --json` |
| Identity | `agent register --name NAME [--role ROLE] [--provider PROVIDER] --request KEY --json` |
| Observe identities/contact/claims | `agent list --json` |
| Start or reconcile | `session start --agent NAME_OR_ID --request KEY --json` / `session resume --session S --request KEY --json` |
| Declare idle or stop | `session idle --session S --request KEY --json` / `session stop --session S --request KEY --json` |
| Create | `ticket create --title TEXT --body-file PATH [--assigned-to NAME_OR_ID] --request KEY --json` |
| Assign within authorization | `ticket assign ID --to NAME_OR_ID_OR_none --expect-revision N --request KEY --json` |
| Find | `ticket list [--state STATE] [--assigned-to NAME_OR_ID_OR_none] --json` |
| Refresh | `ticket show ID --json` |
| Acquire | `ticket claim ID --session S --expect-revision N --request KEY --json` |
| Progress | `ticket note ID --session S --claim C --body-file PATH --request KEY --json` |
| Submit | `ticket submit ID --session S --claim C --expect-revision N --summary-file PATH --evidence-file PATH --qa-file PATH --request KEY --json` |
| Owner release | `ticket release ID --session S --claim C --expect-revision N --reason TEXT --request KEY --json` |
| Problem | `problem add [--ticket ID] --session S --body-file PATH --request KEY --json` |
| Preserve and extend report | `problem append PROBLEM_ID --session S --body-file PATH --request KEY --json` |
| Read reports | `problem list --json` |

A problem-add body is JSON with `summary`, `expected`, `actual`, and optional `correction` and `evidence` text. A problem-append body is a text note. Neither requires a ticket claim nor changes a ticket revision. Preserve the original report even after fixing the problem.

Only explicitly instructed human actions use:

```text
ticket reject ID --human --expect-revision N --reason TEXT --request KEY --json
ticket accept ID --human --expect-revision N --request KEY --json
ticket release ID --human --expect-revision N --reason TEXT --request KEY --json
```

After rejection, refresh and claim the ready ticket anew; the submitted claim remains inactive. A retry may return a historical claim result: check whether it is still active. `session resume` reconciles a known, non-stopped session and its current claim. It never starts another session or revives a stopped one. Idle/stop require no active claim. Availability and `last_seen_at` are observations, not host process liveness.

## Worked delivery handoff

APP-12's declared criteria cover its implementation worktree. Its submission summary should have these concrete slots, filled with actual values:

```text
Ticket: APP-12
Scope and criteria completed: parser accepts X and rejects Y; CLI prints Z.
Tested source commits: <full commit A>, <full commit B>
Validation: <criterion/check, outcome, tested setup, evidence link with executor provenance>
Remaining human QA: <reproducible steps and expected observations>
Delivery: direct landing per project preference; or optional PR <URL> containing APP-12 and APP-13.
Next action: <remaining integration/QA action and owner, only if needed>
```

PRs are optional, may group multiple coherent tickets, and do not determine ticket state. Read the project's delivery preference; do not infer merge permission from assignment. Implementation acceptance can cover tested worktree commits when its criteria say so.

For a feature or bug, keep review, validation, landing, and combined behavior checks as checklist items or status on that ticket. Record included source commits, exact landed target SHA (including squash/rebase results), validation commands and outcomes, and combined manual QA there. Compare the current target SHA with that recorded SHA before treating the evidence as current; a newer target needs fresh validation. Create a separate ticket only when combined behavior is a distinct user-visible feature or defect, not merely a workflow stage. Human acceptance remains a separate explicit action where the ticket policy requires it.

## Prepared-work projects

Read [workflow commands](workflow-commands.md) for draft preparation, versioned authorization, dependencies, blocking, and non-human validation. The human accept/reject examples above apply only to tickets whose recorded policy requires or permits those explicit user actions. Managed startup uses [team commands](team-commands.md); problem processing uses [retrospective commands](retrospective-commands.md).

Read the bootstrap before discovery and use its installation access preference. An established Windows Codex `require_escalated` preference applies to every invocation of that exact executable/home, including version and access probes. A successful call does not change the next process's permissions. For version only, use `--home DATA_HOME --version` without `--project`. Preserve the configured home on access failures; `REGISTRY_UNAVAILABLE` or `DISCOVERY_UNAVAILABLE` is unknown registration, never permission to create a fallback store.
