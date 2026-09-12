# Conductor

A small local ticket tracker for coding agents. One SQLite database coordinates work across Git worktrees; a portable CLI lets agents claim tasks, record progress and workflow problems, and submit work for human review.

The first increment is a CLI tracer: prove the complete working loop before adding a web application. Development progress is on the [work board](PROGRESS.md).

## Getting started

Follow the [quickstart](docs/quickstart.md) to build or install the executable and the Codex/Claude Code work skill. Windows amd64 is the first runtime target. Linux binaries can be built now; Linux runtime acceptance is still pending.

Setup installs a small user-level instruction and skill, preserving existing settings. It does not register any repositories. Run `conductor init --prefix APP --request onboarding/init/1 --json` in each repository you want to use. Linked worktrees share that registration; a separate clone does not automatically opt in. In other projects the agent performs only a read-only registration check and continues without the Conductor workflow. No hooks or repository instruction files are installed.

## What the tracer covers

- Stable agent identities, separate sessions, assignment, and transactional claims.
- Tickets moving through ready, in progress, review, and done; explicit human QA and recovery.
- Retriable mutations with caller-retained request IDs, revision checks, and stale-claim protection.
- Append-only workflow problem reports and follow-ups, including reports without a ticket.
- One database outside the checkouts, shared by processes on one computer.

Claims coordinate tracker ownership; agents still use isolated Git worktrees for implementation. Sessions report contact and declared state, not inferred process liveness. PRs are optional and may group several tickets.

The next increments add dependency tracking, optional checklists and parent tickets, retrospective processing, autonomous worker/conductor skills, and a React + MUI Kanban portal. Markdown ticket rendering and linked-document sidebars come after the initial board. See the [design](docs/superpowers/specs/2026-09-12-conductor-design.md) and [tracer implementation plan](docs/superpowers/plans/2026-09-12-conductor-tracer.md).

Actual host checks and limitations are recorded in [host validation](docs/testing/live-host-checks.md). Configuring filesystem access is separate from proving that a fresh agent session can use it.
