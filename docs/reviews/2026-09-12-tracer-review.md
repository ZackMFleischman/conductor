# CLI tracer implementation review

The final source review approved the tracer after the correction committed as `0824531` and integrated as `3143c69`. The reviewed scope is the CLI, shared SQLite store, installer, work skill, and subprocess tests. Live-host acceptance is recorded separately in [tracer acceptance](../testing/tracer-acceptance.md).

The architecture remains a local CLI over one shared database. There is no service, hook, process supervisor, required PR, frontend, dependency engine, or retrospective processor in this increment.

## Findings resolved

- Ticket mutations now update session contact atomically, and claims record the selected worktree on both claim and session. Exposed claim activity is accurate.
- Agent UUIDs take precedence over names; UUID-shaped names are reserved. Read commands use read-only database opening.
- Problem reports accept project-scoped ticket keys as well as UUIDs, retain original fields, and bound appended-note output with explicit truncation.
- Setup preview excludes private configuration snapshots. TOML edits preserve unrelated semantic values; interrupted removal is recoverable without losing later personal edits.
- Windows setup avoids adding incompatible global writable roots in unelevated or unknown modes and reports normal command approval explicitly.
- Git discovery failures now report unknown registration with diagnostics instead of silently opting a registered project out.
- The work skill selects and enters the intended worktree before claiming. Invalid problem input returns the documented usage exit code.
- Integration tests assert distinct appended text, attribution, persisted session state, and valid contact values, in addition to concurrency, replay, recovery, and rollback.

No unresolved Critical or Important source-review findings remained after the scoped re-review. The full combined tests and vet passed on clean commit `8096420`; portable Windows and Linux candidate builds succeeded. Subsequent fresh Codex and authenticated Claude acceptance passed. Linux runtime and human QA remain separate gates.

Live upgrade testing also established a tracer limitation: setup repairs the recorded installation. A newer bundled skill requires the documented owned-integration removal/reinstallation sequence. That sequence was exercised successfully against the real installation, preserving tracker data and unrelated settings; both installed skill hashes matched the reviewed source afterward.

## Implementation decisions

The user authorized autonomous implementation and parallel isolated work. These decisions were recorded during execution:

| Decision | Reason and tradeoff |
| --- | --- |
| Run independent tasks in parallel worktrees | Follows the user's request; serial integration and review are needed afterward. |
| Use an ignored local Go toolchain | Avoids global toolchain changes; consumes local development disk space. |
| Extract exact T0–T5 plan sections for agent briefs | The skill's helper expected different heading syntax; extraction must preserve the original plan text. |
| Use `local-user` for sessionless actions | Matches the cooperative same-OS-user model; the human flag is workflow intent, not authentication. |
| Permit registration before the first Git commit | Supports new repositories; observed HEAD can initially be empty. |
| Prepare installer work alongside the foundation | No shared-file edits; final CLI integration still waits for the foundation. |
| Prepare packaging before integration finishes | Shortens elapsed time; build preparation does not count as runtime acceptance. |
| Add minimal help/version and use a TOML parser | Makes installation inspectable and preserves configuration semantics; adds one parser dependency. |
| Reserve UUID-shaped agent names and prioritize exact IDs | Prevents ambiguous identity selection; restricts a small class of names. |
| Prepare process tests before all commands land | Establishes expected failing tests early; final green waits for reviewed feature integration. |
| Install under the user-local Conductor directory | Avoids administrator installation and disposable worktrees; relocation requires integration removal/reapply. |
| Use normal command approval for incompatible Windows sandbox modes | Avoids disrupting unrelated projects; even a read-only tracker check may need approval on this host. |
| Document removal/reinstall for skill upgrades | Keeps this tracer installer small; upgrades require an extra reversible step. |
| Publish a feature branch while keeping main unchanged | Makes the work available in the provided repository; final landing remains a separate project decision. |
