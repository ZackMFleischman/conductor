# Conductor CLI quickstart

This release is a local CLI with work, planning, single-job worker, orchestrator, retrospective, and workflow-improver skills. Git is required for repository/worktree discovery. The team skills use native host agent tools; there is no process launcher, daemon, hook, or React/MUI portal. Windows amd64 is the initial runtime target; Linux CLI usage remains provisional until runtime acceptance there.

## Install and inspect setup

Place the built Windows executable in a stable directory, for example `%LOCALAPPDATA%\Conductor\bin\conductor.exe`. A packaged executable includes its skill files; it does not need the source repository at runtime. To build from source with the pinned Go toolchain, run `go build -o conductor.exe ./cmd/conductor` and copy that executable to the chosen location.

PowerShell requires the call operator for quoted executable paths:

```powershell
$Conductor = Join-Path $env:LOCALAPPDATA 'Conductor\bin\conductor.exe'
& $Conductor setup --agents codex,claude --json
& $Conductor setup --agents codex,claude --apply --json
& $Conductor doctor --probe-write --json
```

Setup previews by default; its JSON edits include paths, prior hashes, actions, and descriptions of the owned changes. Personal configuration bytes remain private to the installer and its local ownership journal. `--apply` computes and applies that plan, checking each prior hash immediately before replacement. No setup command initializes the tracker. Applying several files is not transactional: a failure reports the completed count; rerun preview to inspect and repair. Install and removal transitions are journaled before file changes so interrupted operations can resume. Ownership manifests protect user-edited files/blocks from overwrite. Resolve a conflict deliberately rather than deleting personal content.

The default Windows data directory is `%LOCALAPPDATA%\Conductor`, containing `conductor.db`. An explicit `--home 'C:\shared data\Conductor'` overrides `CONDUCTOR_HOME`, which overrides the platform default. Use the same home for setup, doctor, and all tracker commands. It is never inferred from the working directory.

Codex uses the actual `CODEX_HOME` (normally `~/.codex`) for global instructions/configuration. Setup selects a nonempty existing `AGENTS.override.md`; otherwise it uses `AGENTS.md` without creating a precedence override. Its seven Conductor skills, including `conductor-onboard`, are copied under `~/.agents/skills/`. Claude uses `CLAUDE_CONFIG_DIR` (normally `~/.claude`) for `CLAUDE.md`, `settings.json`, and its `skills/` directory. Existing instructions, hooks, and unrelated settings are preserved. Setup merges only the exact Conductor data directory into compatible Codex `sandbox_workspace_write.writable_roots` and Claude `permissions.additionalDirectories`; the Windows exception below avoids disrupting unrelated projects. Setup copies the canonical workflow, observer, and orchestration guidance together. After an upgrade, start fresh agent contexts so they load the installed guidance; project instructions and safety boundaries still apply.

Start fresh host sessions after applying. Configuration is not proof of access: run `doctor --probe-write --json` inside each selected host, then confirm the read-only context bootstrap and native skill discovery. Inspect individual probe outcomes separately from configured paths. A managed sandbox may still deny database/WAL/sidecar access; report that limitation. Do not disable sandboxing or create a fallback registry. The host versions targeted for acceptance are Codex CLI 0.154.0 and Claude Code 2.1.217; synthetic setup tests alone do not establish host acceptance.

On Windows, Codex CLI 0.154.0 with `[windows] sandbox = 'unelevated'` can report `cannot enforce split writable root sets directly; refusing to run unsandboxed` for a separate shared data directory. This can affect ordinary commands in unrelated projects too. Setup therefore installs the bootstrap/skill without adding a global writable root when Windows sandbox mode is unelevated, missing, or unknown. It reports `access.codex.mode = command_approval` and a warning; existing settings and roots remain intact. Explicit elevated mode retains the narrow root configuration. Setup never changes sandbox mode. Use the host's normal per-command approval flow when available and permitted, then rerun the exact probe; otherwise report blocked access. Never disable protections or silently switch to a per-worktree database. Claude's live checks also require an authenticated host session.

## Register and work a ticket

Projects opt in individually. Setup alone does not register a project: `init` registers its Git common directory in the shared database. All linked worktrees use that registration; separate clones require their own `init`. The installed bootstrap performs a read-only context check in other projects and skips the work skill when they are unregistered. An inaccessible registry is an error, not proof that a project is unregistered.

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

## Submit, review, and recovery

Prepare `summary.md` with changed scope and exact criteria completed; `evidence.md` with tested source commits and commands/outcomes; and `qa.md` with reproducible manual steps. Keep review, validation, landing, and combined checks on the feature or bug ticket as checklist items or status. Record optional PR links/grouped tickets in ordinary notes. A PR is optional and can contain multiple tickets. Create a separate ticket only for a distinct user-visible feature or defect. Refresh checks if the target SHA changes.

```powershell
& $Conductor ticket show APP-1 --json
& $Conductor ticket submit APP-1 --session SESSION_UUID --claim CLAIM_UUID --expect-revision CURRENT_REVISION --summary-file '.\summary.md' --evidence-file '.\evidence.md' --qa-file '.\qa.md' --request 'work/submit/1' --json
```

Submission releases ownership and enters review. New plain tickets support agent decisions without planning or team setup; see [foundation commands](reference/ticket-commands.md) for structured accept/reject examples, hierarchy and dependencies. The examples below show the retained legacy human-QA path. Optional workflow policy can require independent-agent, automated or human validation; see [workflow commands](reference/workflow-commands.md). The worker must not issue `--human` to impersonate human acceptance. After a human tests a legacy result, rejection returns it to ready:

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

## Upgrade or remove integration

Replace the executable at its stable installed path, then rerun setup. Setup upgrades unchanged owned skill files in place and adds newly bundled skills. It journals the old and new bytes before changing files so interrupted upgrades can be repaired. User-edited files remain conflicts, not silent overwrites. Preview before applying and start fresh host sessions afterward:

```powershell
& $Conductor setup --agents codex,claude --json
& $Conductor setup --agents codex,claude --apply --json
```

Version 0.2 uses database schema 3 and skill contract `layers-v1`. Stop old CLI processes before the first write with the new executable. That write creates a consistent `conductor.db.pre-v3-*` backup before migrating an existing version-1 or version-2 store. Read-only discovery accepts those older versions without migrating. Older binaries refuse newer schemas. Keep the backup; restore only with all clients stopped and consistent handling of SQLite sidecars. New empty stores need no migration backup. Existing tickets retain their recorded validation restrictions; upgrading does not silently authorize agent acceptance of legacy human-QA work.

For removal without reinstalling:

```powershell
& $Conductor setup --remove --agents codex,claude --json
& $Conductor setup --remove --agents codex,claude --apply --json
```

This removes only unchanged Conductor-owned integration and preserves tracker data and unrelated user configuration. User-edited owned content produces a conflict for manual reconciliation. Restart host sessions after removal. Remove the executable separately if desired; keep the data directory to retain history.

## Onboard an existing project

`init` registers a Git repository and leaves its files untouched. It does not read existing plans or create a backlog. After installing the current skill bundle, ask an agent in the project's checkout:

> Use conductor-onboard to bring this project under Conductor with prefix APP. Preserve its docs and decisions, reconcile existing implementation and acceptance evidence, and import remaining approved work. Keep existing workflow policy; use ordinary tracking if none is configured. Record source-to-ticket coverage and a resumable checkpoint. Stop after the onboarding handoff; do not start implementation.

For an assessment without changes, ask for a dry run. The skill identifies authoritative sources, current commitments, implementation versus acceptance gaps, future ideas, and conflicts that require a user decision. Existing plans stay in place; the onboarding record links them to tickets or explains why no ticket is needed. It checks all ticket pages and reconciles interrupted requests before creating more work.

If planning gates or independent validation are wanted, specify that policy before import: new tickets inherit the policy in force when created. Onboarding does not retroactively apply policy to existing tickets, certify historical work as newly accepted, or start a team. `conductor-plan` handles preparation of the resulting work; `conductor-orchestrator` handles an explicitly authorized execution run.

## Optional managed workflow

Project registration, workflow policy and managed teams are separate. Ordinary registered projects have tickets, hierarchy, dependencies, blockers and agent decision evidence. Optional policy adds preparation, execution authorization and result-validation requirements; it can operate with a standalone agent. The user can delegate covered work fully to agents. Human QA is required only by retained policy, including legacy tickets. See [workflow commands](reference/workflow-commands.md), [team commands](reference/team-commands.md), and [retrospective commands](reference/retrospective-commands.md) for payloads and command syntax.

Ask an agent to use conductor-orchestrator for an authorized team and scope. It reconciles existing roles and approvals, starts the workflow improver where enabled, plans work using whatever policy is configured, and starts fresh single-ticket workers as independent tasks become ready. All roles count against host limits. Workers may create discoveries; policy determines whether those tickets need preparation before execution. Installing skills or assigning a ticket alone starts no background process. Retrospective analysis and improvement tickets also work without a team.

A host with no supported delegation can run bounded planning/retrospective and ordinary ticket work. It must report that managed agents were not launched. Linux execution and provider-specific native management are supported only where separately validated; sharing tickets across Codex and Claude does not imply one can control arbitrary processes of the other.

### Established Windows Codex command access

When this installation has an established per-command elevation requirement, preview `setup --agents codex --command-access require_escalated --json` with the installed executable and configured `--home`, then add `--apply` to install the owned changes. This records the exact executable/home preference before the bootstrap's first discovery command. It changes no sandbox mode, execution policy, or unrelated command permissions. The option supports Windows Codex only; configure other hosts separately.

Omitting `--command-access` during upgrades preserves the recorded preference. Explicit `--command-access host-default` resets it. Omit the option for `--remove`. Updates journal old and desired owned bootstrap bytes, preserve surrounding personal text, and refuse edited or duplicated owned blocks. After interruption, rerun the same preview (or omit the preference to finish its recorded value) before choosing a different preference. Removal also repairs from its journal. Start a fresh host context after adoption and verify its first matching invocation follows the installed preference; source tests alone do not prove that host behavior.
