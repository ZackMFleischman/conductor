# CON-18: parent subtree and compact blocker verification

Logical parent: CON-8. Dependency: CON-17 delivered density/search implementation. Integration proposal: CON-19, assigned to coordinator-1 for CON-4, extending CON-13. Fixture-only; live integration remains pending.

Source commit: `fc5783a1813d033ba20966197a571398bd10394f` on `feat/portal`.
Base: `a3772fbf3809175bc4558b347b4ccd4b2a4ca5e0`.
Worktree: `C:/Users/zFlei/repos/conductor/.worktrees/portal`.
Agent: portal-coordinator-1 (`769fc4fd-2725-4930-94c8-5a9f6d977747`).
Session: `ae1120f5-1f55-4e4a-bdf7-a9d33c5f0de6`.

## Completed behavior

All parents is the single filter label. Selecting any parent includes that ticket if present and all descendants at every depth, using complete nearest-parent-to-root ancestor chains. Intermediate parents need not be visible board rows. No parent matches an explicitly empty chain. Search and assignee filters further narrow results; Clear filters resets all three.

Muted clickable root titles sit beside assignees; tooltips expose full ancestor paths. Blocked by and related ticket keys now share one compact row above the complete reason. Full descriptions remain visible. This supersedes CON-18/CON-19's original separate epic/direct-parent proposal; the definitive pending contract is api-contract.md at the source commit above.

## Verification

On 2026-09-12 at 18:13 America/Los_Angeles, using the source committed above:

- `npm test` from web/portal: PASS, 4 files / 18 tests, 10.47 seconds.
- `npm run build`: PASS, TypeScript and Vite, 914 modules, JS 499.42 kB / gzip 157.49 kB.
- `git diff --check`: PASS before commit.
- Four group tests cover any-depth descendants with missing intermediate rows, inclusion of the selected parent itself, combined filtering/reset, opaque IDs colliding with control values, and clickable labels/tooltips.
- Independent review found one stale JSON contract example; corrected to ancestors. No code defects found. Correction preserved in CON-P26.

Browser at http://127.0.0.1:4173/:

- Desktop 1440x900: All parents visible, all five lanes, four Ready cards and three each In progress/Blocked visible; full descriptions and compact inline blocker key.
- Search & discovery selects DEMO-105 (child) and DEMO-112 (grandchild); Match rendering selects only DEMO-112. The fixture's intermediate parents are references, not board cards.
- Clear filters restores all 12 cards.
- Mobile 390x844: compact controls wrap, root labels remain unobtrusive; document width 375px, no horizontal page overflow.
- No browser warning/error entries after 01:10 UTC. Viewport override reset and preview retained.

Screenshots: [desktop](evidence/groups-desktop.png), [subtree filter](evidence/group-filter.png), [mobile](evidence/groups-mobile.png).

## Human QA and integration handoff

Run npm test and npm run build in web/portal; start npm run dev -- --port 4173 --strictPort. Confirm All parents default, select Search & discovery then Match rendering, combine search and assignee, and clear. Hover or keyboard-focus a root label to inspect the full path; click to filter. Verify compact blocker key and complete reasons on desktop/mobile. Fixture data badge stays visible.

CON-19 / CON-4: review the revised required ancestors chain and authorization semantics in api-contract.md; it supersedes separate epic/parent fields. An agreed live contract, runtime validation, real adapter and actual-service browser evidence are separate future work. No live success claimed from these fixtures.

Submit CON-18 for explicit human QA; do not self-accept or merge. Only portal-owned files changed. No core/schema/install/root build/PROGRESS.md or case-study work. A subsequent documentation-only commit records this evidence.
