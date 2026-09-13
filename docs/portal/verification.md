# Portal increment 1 verification and handoff

## Source and tracking

- Branch: `feat/portal`; worktree: `C:/Users/zFlei/repos/conductor/.worktrees/portal`.
- Starting commit: `c5e26b4c47176a78d243d03d21fe367ae3c5aaaf`.
- CON-11 contract/fixture source: `54fe2eabcaefe94e9f9a8fd327198c36b1e7713f`.
- CON-12 compact board and browser evidence: `58a8031af48bb4b6ccd8cef75da2b32e8cd1c779`.
- Stable identity: portal-coordinator-1 (`769fc4fd-2725-4930-94c8-5a9f6d977747`). Session: `954adbad-0403-4d49-ad5a-0233d7e5ab2f`.
- Logical parent: CON-8. CON-12 depends on CON-11 implementation. CON-13 is a proposal assigned to coordinator-1 for CON-4, requesting agreement on the read boundary.

## Automated evidence

Executed from `web/portal` at source commit `58a8031af48bb4b6ccd8cef75da2b32e8cd1c779` on 2026-09-12, approximately 17:45 America/Los_Angeles (2026-09-13 UTC):

| Command | Result |
| --- | --- |
| `npm test` | PASS: 2 files, 10 tests, 0 failures; 8.76 seconds. |
| `npm run build` | PASS: TypeScript check and Vite production build; 911 modules; JS 464.63 kB / 145.53 kB gzip. |
| `git diff --check` | PASS. |
| `git diff --name-only c5e26b4c47176a78d243d03d21fe367ae3c5aaaf HEAD` | Every changed file is under web/portal or docs/portal. |

Node 24.12.0, npm 11.6.2. Exact dependencies are recorded in the local package/lockfile. Installation audit reported zero vulnerabilities at installation time.

Tests cover fixture identity, all display statuses, full descriptions, unassigned and blocked records, isolated snapshots, abort handling, UI rendering, combined case-insensitive search/assignment, clearing filters, real empty board vs filtered empty results, safe error/retry without fixture fallback, pending load cancellation, text-only rendering, missing-detail fallbacks and opaque assignee IDs that collide with special option names.

Red-to-green evidence: empty fixture assertion failed before implementation; six UI cases failed against the empty component; reviewer regression for assignee id `all` failed with three cards instead of one before the tagged-value fix. Independent contract review found no issues. Independent UI review identified the collision and confirmed its correction.

## Browser evidence

Inspected the running Vite app at `http://127.0.0.1:4173/` in the in-app Chromium browser. Final density revisions include both rounds of user feedback: remove tall introductory sections, remove redundant fixture explanation and result-count line, and shorten controls.

| Scenario | Observation |
| --- | --- |
| Desktop 1440 x 1000 | All five lanes and eight fictional tickets visible; full descriptions, assignments and warm blocker panels. Board begins around y=145px. |
| Search `WAITING` | Only DEMO-105 remains; blocker reason matches case-insensitively. |
| Add `Unassigned` filter | Explicit “No tickets match these filters”; no cards. |
| Clear filters | All eight cards return and controls reset. Rechecked after final control changes. |
| Select `docs-agent` | Only DEMO-102 and DEMO-104 remain. Rechecked after owner-value fix. |
| Keyboard | Tab from search focuses Assignee; lane group accepts keyboard focus for horizontal scrolling. |
| Intermediate 900 x 800 | Page width stays within viewport; overflow is contained in the board (837px client / 1268px content). |
| Mobile 390 x 844 | Lanes stack, compact controls fit together, no horizontal page overflow (375px document width with scrollbar); board begins around y=177px. |
| Console | No new warnings/errors after supported MUI prop fixes; earlier warnings were retained as historical diagnostic evidence. |

Screenshots: [desktop](evidence/desktop.png), [mobile](evidence/mobile.png), [empty filters](evidence/empty-filter.png). Loading/error/empty-source and hostile-string behavior are covered by component tests using injected sources, not by an actual server.

## Human QA and integration boundary

From web/portal, run `npm ci`, `npm test`, `npm run build`, then `npm run dev -- --port 4173 --strictPort`. Check the fixture/read-only badges, full descriptions, assignment and blocker explanations. Exercise search and assignment together, then clear. Check desktop and narrow widths; confirm no editing, claiming or drag-to-change-state controls exist.

Review the proposed [API contract](api-contract.md) in CON-13 / CON-4. Agree transport, authorization and core lifecycle mapping before a live adapter is implemented. A later integration ticket must bind combined behavior to the exact landed core and portal commits. Fixture success is **not** live integration success.

CON-11 and CON-12 require explicit human QA; no worker acceptance was performed. CON-8 stays open for later increments. Markdown details and the rendered linked-artifact sidebar are not implemented. No core merge, schema/installation changes, or case-study work occurred.

Workflow problems and corrections are preserved in CON-P17 and CON-P21, including sandbox access, initial validation mistakes, user density feedback and the review fix. The follow-up evidence commit changes only portal documentation.
