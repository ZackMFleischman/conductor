# Portal increment 1: design and implementation plan

Goal: a runnable, read-only React + MUI Kanban board, explicitly showing fixture data.

## Design

A quiet off-white workspace with a dark ink header, teal project mark and five status lanes. Full descriptions remain visible; blockers get a warm tinted explanation, while assignment is a separate text label. Status uses words and color together. Summary counts provide context; search and an assignee select only filter the local view. The board stacks on phones and scrolls horizontally at intermediate widths to preserve readable cards.

Architecture: `src/board.ts` owns presentation types, `src/data/fixtureBoardSource.ts` supplies fictional records through `BoardSource.load(signal)`, and React components consume only that interface. Source identity is separate from the asynchronous snapshot so fixture labeling persists during errors/loading. Stable ticket IDs leave room for a later detail selection. No router, Markdown renderer, artifact retrieval, mutation UI, SQLite binding or server API is needed now.

Alternatives considered: a CLI-output-shaped model would couple the UI to refactoring internals; a server now would cross the ownership boundary. The small frontend snapshot interface lets core independently map its lifecycle later.

## Execution (CON-8 logical parent)

- CON-11: define board types and fixture adapter; test all statuses, blockers, unassigned records, isolation between loads and cancellation; document the proposed contract in api-contract.md; commit and submit for human QA.
- CON-12 depends on CON-11 implementation: build MUI board/cards/filter controls and explicit async states. Test filtering and failures at the UI boundary. Build, inspect desktop and narrow browser views, record evidence, commit and submit for human QA.
- CON-13: proposal assigned to coordinator-1 / CON-4. Agree contract before any live integration implementation. Fixture acceptance is separate from live integration acceptance.

All edited files and package configuration stay in web/portal/** or docs/portal/**, on feat/portal. Do not merge. No core, root build, schema, installation, skill, PROGRESS.md or case-study work.

## Verification

Run `npm test` and `npm run build` from web/portal. Browser QA checks all five lanes, descriptions, explicit assignment/unassigned text, blockers, fixture labeling, case-insensitive search, combined assignee filtering, clear filters, empty results, keyboard navigation and a narrow viewport. Error/loading behavior is also exercised with injected sources in component tests. Exact source commits and reproducible manual QA belong in verification.md and tracker submissions.
