# Portal backend implementation plan

**Goal:** Read the existing Conductor database through a small local API and notify the portal when separate agents change it.

**Architecture:** One existing Go binary gains `serve`; `internal/portalapi` owns the read model and HTTP/SSE transport. SQLite remains authoritative; no migrations or workflow changes.

**Spec:** [Backend contract](../../backend/api-contract.md), resolving CON-19 and CON-13.

**Constraints:** Go 1.27.1 and existing dependencies only; Windows first and Linux build; loopback/same-origin; frontend-owned paths untouched; explicit errors rather than fixture fallback. Use real SQLite tests. Root owns HTTP/CLI/integration; a separate worker may own the read model in its own worktree after the contract review.

1. **CON-19 contract:** independently review complete ancestry, state mapping, read consistency, external writers and reconnect behavior. Commit contract and record result; release the prerequisite before API implementation.
2. **Read model (parallel after shared types):** add `internal/portalapi/board.go` and tests. `DBReader{DB *sql.DB}` implements `Projects(context.Context) ([]Project,error)` and `Board(context.Context,string) (Board,error)`. Board fields match the contract, revision hashes the snapshot plus project event watermark. Red tests seed real schema-3 data, multiple projects, nested parents, dependencies/policies/deferrals, then implement transactional queries and fail-closed graph validation. Commit isolated worker output.
3. **HTTP/SSE (parallel with read model):** root adds shared `model.go`, `server.go`, `events.go` and tests against a Reader interface. First prove missing handlers fail, then implement GET-only routes, local request checks, safe errors, consistent snapshot output, immediate/refreshed SSE signals, cancellation, write deadlines and bounded streams. Do not hold SQL transactions while streaming.
4. **Runnable CLI and integration (after 2+3):** new self-registering `internal/cli/serve.go` avoids edits to shared CLI dispatch. Validate arguments before opening existing read-only schema3 store, serve optional trusted static build, log bound address, shut down on cancellation. Add real SQLite/external-writer SSE tests and CLI help/cancellation tests. Run `go test ./...`, `go vet ./...`, Linux build, independent code review, and a real installed-CLI mutation against a separate fixture registry.
5. **Frontend integration (CON-22, parallel with 2–4):** following the user's authorization, merge main's portal into this branch and delegate `web/portal/**` and `docs/portal/live-integration.md` to an isolated worker. Consume the agreed API through a strict decoder, project picker and SSE invalidation with coalesced refresh. Preserve explicit fixture mode. Independently review the adapter, then validate the combined build in a browser against CLI-created tickets, ancestry filters and offline/reconnect recovery.
6. **Handoff:** commit evidence and exact source commits in `docs/backend`. Do not claim fixture screenshots prove live integration. Leave existing legacy acceptance policies intact.
