# Conductor tracer quickstart

This release is a local CLI and one work skill. Git is required for repository/worktree discovery. The React/MUI portal, autonomous worker/orchestrator loops, and hooks are not included. Windows amd64 is the initial runtime target; Linux CLI usage remains provisional until runtime acceptance there.

## Install and inspect setup

Place the built Windows executable in a stable directory, for example `C:\Program Files\Conductor\conductor.exe`. A packaged executable includes its skill files; it does not need the source repository at runtime. To build from source with the pinned Go toolchain, run `go build -o conductor.exe ./cmd/conductor` and copy that executable to the chosen location.

PowerShell requires the call operator for quoted executable paths:

```powershell
$Conductor = 'C:\Program Files\Conductor\conductor.exe'
& $Conductor setup --agents codex,claude --json
& $Conductor setup --agents codex,claude --apply --json
& $Conductor doctor --probe-write --json
```

Setup previews by default; its JSON edits include paths, prior hashes, actions, and descriptions of the owned changes. Personal configuration bytes remain private to the installer and its local ownership journal. `--apply` computes and applies that plan, checking each prior hash immediately before replacement. No setup command initializes the tracker. Applying several files is not transactional: a failure reports the completed count; rerun preview to inspect and repair. Install and removal transitions are journaled before file changes so interrupted operations can resume. Ownership manifests protect user-edited files/blocks from overwrite. Resolve a conflict deliberately rather than deleting personal content.

The default Windows data directory is `%LOCALAPPDATA%\Conductor`, containing `conductor.db`. An explicit `--home 'C:\shared data\Conductor'` overrides `CONDUCTOR_HOME`, which overrides the platform default. Use the same home for setup, doctor, and all tracker commands. It is never inferred from the working directory.

Codex uses the actual `CODEX_HOME` (normally `~/.codex`) for global instructions/configuration. Setup selects a nonempty existing `AGENTS.override.md`; otherwise it uses `AGENTS.md` without creating a precedence override. Its personal skill is copied to `~/.agents/skills/conductor-work`. Claude uses `CLAUDE_CONFIG_DIR` (normally `~/.claude`) for `CLAUDE.md`, `settings.json`, and `skills/conductor-work`. Existing instructions, hooks, and unrelated settings are preserved. Setup merges only the exact Conductor data directory into Codex `sandbox_workspace_write.writable_roots` and Claude `permissions.additionalDirectories`.

Start fresh host sessions after applying. Configuration is not proof of access: run `doctor --probe-write --json` inside each selected host, then confirm the read-only context bootstrap and native skill discovery. Inspect individual probe outcomes separately from configured paths. A managed sandbox may still deny database/WAL/sidecar access; report that limitation. Do not disable sandboxing or create a fallback registry. The host versions targeted for acceptance are Codex CLI 0.154.0 and Claude Code 2.1.217; synthetic setup tests alone do not establish host acceptance.

On Windows, Codex CLI 0.154.0 with `[windows] sandbox = 'unelevated'` can report `cannot enforce split writable root sets directly; refusing to run unsandboxed` for a separate shared data directory. Setup does not change that sandbox mode. Use the host's normal per-command approval flow when available and permitted, then rerun the exact probe; otherwise report blocked access. Never disable protections or silently switch to a per-worktree database. Claude's live checks also require an authenticated host session.

## Register and work a ticket

Run these from a Git repository/worktree. Substitute returned UUIDs and revisions for placeholders. Every database mutation needs a unique caller-retained request key; retries of one operation reuse its exact key and payload. These examples use distinct descriptive keys once; choose new keys for a new run.

```powershell
& $Conductor init --prefix APP --request 'onboarding/init/1' --json
& $Conductor context --if-registered --json
& $Conductor agent register --name 'worker-1' --provider 'codex' --request 'onboarding/agent/1' --json
& $Conductor session start --agent 'worker-1' --request 'onboarding/session/1' --json
```

Retain the returned stable `agent_id` and `session_id`. An identity may have multiple sessions; each session can own one active claim. An old `last_seen_at` is not proof the process stopped. Prefer an isolated worktree for implementation; intentional integration/serial work may use the shared checkout and acknowledge its warning.

Write a ticket description to `ticket.md` with the exact scope and acceptance criteria, then create assigned work:

```powershell
& $Conductor ticket create --title 'Implement the requested change' --body-file '.\ticket.md' --assigned-to 'worker-1' --request 'work/create/1' --json
& $Conductor ticket list --assigned-to 'worker-1' --state ready --json
& $Conductor ticket show APP-1 --json
& $Conductor ticket claim APP-1 --session SESSION_UUID --expect-revision 1 --request 'work/claim/1' --json
& $Conductor ticket note APP-1 --session SESSION_UUID --claim CLAIM_UUID --body-file '.\progress.md' --request 'work/note/1' --json
```

Use actual identifiers/revisions from responses; notes advance revision. Unassigned tickets are claimable within scope; assigned tickets require the matching identity. Do not change assignment simply to claim another worker's ticket. Resume an existing session after a lost response:

```powershell
& $Conductor session resume --session SESSION_UUID --request 'work/resume/1' --json
```

Record problems without erasing earlier observations. `problem.json` contains:

```json
{"summary":"Validation needed a missing fixture","expected":"Documented test command runs","actual":"Fixture was missing","correction":"Added the fixture","evidence":"Recorded command output"}
```

```powershell
& $Conductor problem add --ticket APP-1 --session SESSION_UUID --body-file '.\problem.json' --request 'work/problem/1' --json
& $Conductor problem append PROBLEM_UUID --session SESSION_UUID --body-file '.\followup.md' --request 'work/problem-followup/1' --json
& $Conductor problem list --json
```

## Submit, human QA, and recovery

Prepare `summary.md` with changed scope and exact criteria completed; `evidence.md` with tested source commits and commands/outcomes; and `qa.md` with reproducible manual steps. Record optional PR links/grouped tickets in ordinary notes. A PR is optional and can contain multiple tickets. Integration uses a separate ticket with included source commits, exact final landed SHA, and combined validation results; refresh checks if the target SHA changes.

```powershell
& $Conductor ticket show APP-1 --json
& $Conductor ticket submit APP-1 --session SESSION_UUID --claim CLAIM_UUID --expect-revision CURRENT_REVISION --summary-file '.\summary.md' --evidence-file '.\evidence.md' --qa-file '.\qa.md' --request 'work/submit/1' --json
```

Submission releases ownership and enters review. Every ticket needs explicit human QA. The worker must not issue `--human` to approve itself. After a human tests the result, rejection returns it to ready:

```powershell
& $Conductor ticket reject APP-1 --human --expect-revision CURRENT_REVISION --reason 'Observed QA failure and reproduction' --request 'human/reject/1' --json
```

Refresh, claim again with a new request key/new claim, append rework evidence, and submit again. When the human verifies its criteria, accept using the latest revision:

```powershell
& $Conductor ticket accept APP-1 --human --expect-revision CURRENT_REVISION --request 'human/accept/1' --json
```

To stop active work normally, save progress, then release and stop. If a worker is abandoned, first inspect its process and uncommitted work. Explicit human recovery invalidates the old claim but does not stop the process or protect its files:

```powershell
& $Conductor ticket release APP-1 --session SESSION_UUID --claim CLAIM_UUID --expect-revision CURRENT_REVISION --reason 'Checkpoint and handoff' --request 'work/release/1' --json
& $Conductor session stop --session SESSION_UUID --request 'work/stop/1' --json
# Alternative recovery, only for the explicit human action:
& $Conductor ticket release APP-1 --human --expect-revision CURRENT_REVISION --reason 'Former worker inspected; recovery authorized' --request 'human/recovery/1' --json
```

Quiet sessions never expire automatically. A stopped session cannot resume; start a new one when appropriate. Claim/revision conflicts require a fresh read and reconciliation, not blind retries.

## Uninstall integration

```powershell
& $Conductor setup --remove --agents codex,claude --json
& $Conductor setup --remove --agents codex,claude --apply --json
```

This removes only unchanged Conductor-owned integration and preserves tracker data and unrelated user configuration. User-edited owned content produces a conflict for manual reconciliation. Restart host sessions after removal. Remove the executable separately if desired; keep the data directory to retain history.
