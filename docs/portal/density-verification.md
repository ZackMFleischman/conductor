# CON-17: density and search verification

User follow-up: smaller default density and text highlighting for search. Logical parent CON-8; depends on the delivered CON-12 implementation. Fixture-only; CON-13 / CON-4 remains the separate API integration proposal.

Source commit: `23141539abc8629f3f03fbd97c72891809eeef11` on `feat/portal`.
Base: `0186364b05184c513de24ae86b644f8df6fc3c31`.
Worktree: `C:/Users/zFlei/repos/conductor/.worktrees/portal`.
Agent: portal-coordinator-1 (`769fc4fd-2725-4930-94c8-5a9f6d977747`).
Session: `ae1120f5-1f55-4e4a-bdf7-a9d33c5f0de6`.

## Completed behavior

Reduced card padding, typography, lane gaps and assignment/status row height. Full descriptions and blockers remain visible. Expanded fictional sample tickets from 8 to 12 so busy lanes contain 3-4 cards.

Search uses a shared, escaped literal matcher for filtering and marking. It trims outer whitespace, ignores case, preserves original text/casing, marks all non-overlapping occurrences in searchable fields, and never interprets HTML. A phrase spanning unrelated fields no longer produces a result with no visible matching phrase.

## Verification evidence

From web/portal on 2026-09-12 at approximately 17:55 America/Los_Angeles, on the source committed above:

- `npm test`: PASS, 3 files / 14 tests, 12.45 seconds.
- `npm run build`: PASS, TypeScript check and Vite build, 913 modules, JS 466.41 kB / 146.06 kB gzip.
- `git diff --check`: PASS before commit.
- Three new tests failed before implementation, then passed: literal punctuation and repeated matches across fields, inert HTML-looking text, and no cross-field phrase matches. Whitespace-only queries and clearing marks also pass.
- Independent review: no actionable defects found in matching, highlight rendering, density or integration with the existing board.

Browser at http://127.0.0.1:4173/:

- 1440x900: all four Ready cards visible (about 136px each); three In progress cards visible (136-153px); three Blocked cards visible. Full descriptions remain rendered.
- Query `  BOARD  `: four cards and five yellow highlights; original lowercase text retained.
- Query `WAITING`: one card with the blocker word Waiting highlighted.
- Clear filters: 12 cards and zero mark elements.
- 390x844: four Ready cards fit in the initial viewport; document width 375px within the 390px viewport, without horizontal page overflow. First card about 119px high.
- Browser viewport override reset; portal left open.

Screenshots: [density desktop](evidence/density-desktop.png), [density mobile](evidence/density-mobile.png), [search highlights](evidence/search-highlights.png).

## Human QA / handoff

Run npm test and npm run build from web/portal, then npm run dev -- --port 4173 --strictPort. Review default density at normal browser zoom. Search board, WAITING, portal-agent and a ticket key; confirm matching text is highlighted and Clear filters removes marks. Check the fixture badge and complete descriptions on desktop/mobile. Special characters and HTML-looking content are verified by injected-source component tests.

Submit CON-17 for explicit human QA. No worker acceptance, merge, core/schema/installation changes, API integration, Markdown/artifact work or case-study work. User feedback/correction is preserved in CON-P25. Subsequent documentation commit contains this evidence only.
