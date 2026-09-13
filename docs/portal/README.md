# Run the Conductor portal

From this worktree, enter `web/portal` and use Node.js 22.12+ (verified with Node 24.12.0 and npm 11.6.2):

```powershell
cd C:\Users\zFlei\repos\conductor\.worktrees\portal\web\portal
npm ci
npm run dev -- --port 4173 --strictPort
```

Open http://127.0.0.1:4173. The server binds to loopback. All displayed DEMO tickets are fictional. No API, database connection, or installed Conductor change is needed.

```powershell
npm test
npm run build
npm run preview -- --port 4174 --strictPort
```

`build` type-checks and creates `web/portal/dist`. `preview` serves that output locally. This package owns its configuration, exact direct dependency versions and lockfile. Root build files are not involved.

## Code map

- `src/board.ts`: frontend-owned types and `BoardSource` interface.
- `src/data/fixtureBoardSource.ts`: explicit cancellable fixture source, returning independent snapshots.
- `src/App.tsx`: asynchronous load states, search and assignment filters, compact workspace shell.
- `src/components/KanbanBoard.tsx`: semantic five-lane board; horizontal scrolling at intermediate widths and stacked lanes below 600px.
- `src/components/TicketCard.tsx`: full plain-text description, display status, assignee and blocker reasons.
- `src/theme.ts`: MUI theme and typography.

The composition root in `src/main.tsx` chooses the fixture adapter. No UI component reads CLI output or SQLite. A live adapter requires the separately tracked contract agreement in CON-13 / CON-4. Markdown details and linked-artifact rendering remain later increments. See [API contract](api-contract.md), [design](design.md) and [verification](verification.md).

Latest follow-up: [CON-17 density and search verification](density-verification.md). The board now has 12 fictional tickets and highlights literal search matches; historical first-increment evidence retains the original eight-ticket observations.
