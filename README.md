# Conductor

A small local ticket tracker for coding agents. One SQLite database coordinates work across Git worktrees; a portable CLI lets agents claim tasks, record progress and workflow problems, and hand off results with review evidence.

The first increment is a CLI tracer: prove the complete working loop before adding a web application. Development progress is on the [work board](PROGRESS.md).

## Getting started

Follow the [quickstart](docs/quickstart.md) to build or install the executable and the Codex/Claude Code work skill. Windows amd64 is the first runtime target. Linux binaries can be built now; Linux runtime acceptance is still pending.

Setup installs a small user-level instruction and skill, preserving existing settings. It does not register any repositories. Run `conductor init --prefix APP --request onboarding/init/1 --json` in each repository you want to use. Linked worktrees share that registration; a separate clone does not automatically opt in. In other projects the agent performs only a read-only registration check and continues without the Conductor workflow. No hooks or repository instruction files are installed.

For an existing project with docs and plans, ask an agent to use [conductor-onboard](skills/conductor-onboard/SKILL.md). It reconciles current scope and implementation evidence, preserves source documents, and maps remaining work into tickets with a resumable coverage record. `init` only registers the repository. See [existing-project onboarding](docs/quickstart.md#onboard-an-existing-project) for a sample request and workflow boundaries.

## What the tracer covers

- Stable agent identities, separate sessions, assignment, and transactional claims.
- Tickets, parent/child scope, dependencies, blockers, result submissions and attributed accept/reject decisions without requiring a team.
- Retriable mutations with caller-retained request IDs, revision checks, and stale-claim protection.
- Append-only workflow problem reports and follow-ups, including reports without a ticket.
- One database outside the checkouts, shared by processes on one computer.

Claims coordinate tracker ownership; agents still use isolated Git worktrees for implementation. Sessions report contact and declared state, not inferred process liveness. PRs are optional and may group several tickets.

Optional workflow policy adds preparation, authorization and human, independent-agent or automated result checks. Managed teams add coordinator recovery, native-host readiness and fresh workers restricted to one assignment. Retrospective processing groups problems and tracks proposed remedies with or without a team. These are layers over the tracker, not prerequisites for basic ticketing; see [foundation commands](docs/reference/ticket-commands.md) and the [layer plan](docs/superpowers/plans/2026-09-12-layer-boundaries.md).

A React + MUI Kanban portal is being developed separately. Markdown rendering, linked-document sidebars and optional checklist UI follow the initial board. The portal consumes a documented API rather than depending on SQLite tables. See the [design](docs/superpowers/specs/2026-09-12-conductor-design.md) for later increments.

Actual host checks and limitations are recorded in [layered workflow acceptance](docs/testing/team-workflow-acceptance.md), including the fresh Windows sandbox, native agents, independent result validation and retrospective audit. Earlier tracer evidence remains in [host validation](docs/testing/live-host-checks.md). Configuring filesystem access is separate from proving that a fresh agent session can use it.
