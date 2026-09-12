# CLI tracer acceptance

Observed on Windows amd64, 2026-09-12. The CLI is being dogfooded; full two-host acceptance remains open until Claude Code can be tested in an authenticated session.

## Verified build

Source: `5a247a6ad2d0ed875e69923ff4e6ba473254097f`, clean checkout, Go 1.27.1.

| Check | Result |
| --- | --- |
| `go test ./... -count=1` | Passed, including real subprocess integration tests |
| `go vet ./...` | Passed |
| Portable Windows amd64 executable | Built with CGO disabled |
| Linux amd64 candidate | Cross-compiled with CGO disabled; runtime not tested |
| Documented quickstart | 26 commands passed, including simulated fixture reject/rework/accept and recovery |
| Go race detector | Unavailable: requires CGO; no GCC or Clang found on this host |

The process tests prove one claim winner under contention, assignment enforcement, stale-owner rejection, duplicate-request recovery after a discarded response, append content/author preservation, and rollback after killing a process holding an uncommitted transaction. They also exercise linked worktrees, an unrelated clone, declared session state, and synthetic setup/removal preserving existing files. They do not infer agent process liveness.

Windows Git fixture execution sometimes failed inside the managed sandbox with a signal-pipe permission error. The same complete suite passed using the normal approved execution path. This is distinct from the installed Codex sandbox limitation below.

The build scripts write `dist/build-metadata.txt` and `dist/SHA256SUMS`. Those identify the exact source and hashes of the current artifacts; a later build replaces them.

## Actual local installation

The verified executable was installed at `%LOCALAPPDATA%\Conductor\bin\conductor.exe`. Its embedded work skill and managed bootstrap were installed for Codex CLI 0.154.0 and Claude Code 2.1.217. The executable needs neither Go nor Node at runtime.

Setup preserved the Codex configuration byte for byte, added only the Conductor data directory to Claude's additional directories, and created the owned instruction blocks and skill files. A second apply reported zero edits. The installed executable passed the SQLite open, WAL, persistence, reopen, and cleanup probe through normal approved execution.

The existing Windows Codex sandbox is `unelevated`. Adding a separate global writable root can prevent ordinary commands from running, including in unrelated projects. Setup therefore reports `command_approval` access mode and leaves the global writable roots unchanged. No sandbox mode was changed or protection disabled. Other Windows modes without established compatibility are treated conservatively; explicit elevated mode retains the narrow-root configuration.

A fresh Codex session received an ordinary README task with no tracker commands in its prompt. It ran the installed bootstrap, retried denied database access using normal approval, discovered the native work skill, created its own identity/session and `HOST-1`, claimed the work, recorded a workflow problem and progress, committed the one-file change, and submitted evidence/manual QA. The persisted ticket is in review with its claim released; the session stopped. No push or PR occurred.

A separate fresh session changed an unrelated repository without creating tickets or identities. Its initial database read was denied and it reported registration as unknown rather than asserting an opt-out; it continued the authorized documentation work without claiming tracking updates. The standalone approved context check in that repository returned `registered=false`. On this Windows sandbox, even a read-only registration check may require normal approval; setup cannot guarantee a silent check. Claude Code is signed out, so its live skill/access behavior is not yet verified. Configuration alone is not runtime evidence.

## Dogfooding

The Conductor repository is registered as project prefix `CON` in the default shared database, outside the worktrees. `CON-1` tracks combined integration, and `CON-2` tracks host validation. Eight real workflow problem reports preserve user corrections, failed attempts, review findings, and workarounds; corrections are appended without overwriting original observations.

The coordinator intentionally uses the primary checkout for serial integration; implementation agents used isolated worktrees. A linked worktree resolves the same registered project. An unrelated repository returns `registered=false` against the same database. Fixture tests that simulate human QA do not accept the real dogfood tickets; those remain subject to human review.

## Remaining acceptance

- Recheck the final corrected work skill and integration-evidence freshness behavior.
- Authenticate Claude Code and exercise its installed skill and shared-data access.
- Run the Linux candidate on Linux before claiming Linux runtime support.
- Human QA of the real dogfood delivery tickets.

The next product increment adds functionality on top of this CLI; no portal, hooks, autonomous worker loops, dependency engine, or retrospective processor is included in this tracer.
