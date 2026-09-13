# Portal ticket details and activity

CON-35 delivers recorded type badges, an accessible large details dialog with full safe Markdown descriptions/summary/evidence/QA, project ticket links, Done batches of20 with Show more/Show all, and a collapsible right activity feed.

The feed starts with the first accepted snapshot, records subsequent observed ticket changes newest first, and retains the latest200 entries in memory. Collapsing or retrying does not discard the feed; opening a different project or reloading starts a new session. Reconnect can combine intermediate changes. This is a live observation feed, not a persistent audit log.

Verification on2026-09-13 by portal-coordinator-20260913 in the main worktree:
- Frontend:60 tests across8 files, TypeScript and production build pass. Includes45-card Done paging, full Markdown modal/link navigation, Escape after navigation, live updates, unchanged refreshes, retry recovery/history, and search phrases across ticket links.
- Backend: portalapi suite and CLI TestServe pass; production Go binary builds. Real SQLite regression checks stored kind, summary, evidence, QA and timestamps.
- Independent reviewer portal_review found highlight-boundary and retry-history bugs; both corrected and re-reviewed without remaining actionable findings.
- Candidate browser against actual CON registry shows stored types, complete summary/evidence/QA, CON-1 to CON-2 modal navigation and Escape dismissal. A real installed-CLI CON-35 progress note appeared in the open activity feed without reload.
- Vite retains a nonfatal bundle-size advisory. No dependency changes or database migration.

Deployment uses a versioned executable/assets directory and the existing loopback7332 backend behind the unchanged7331 gateway. Exact commit and deployment observations are recorded in CON-35.
