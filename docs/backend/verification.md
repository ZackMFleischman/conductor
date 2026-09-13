# Live portal verification

Validated on Windows on 2026-09-12 using the combined `codex/portal-api` branch. The service reads SQLite; fixture tickets below were created and changed by the separate installed CLI, not injected into React.

## Source

| Component | Commit |
| --- | --- |
| Main portal/core baseline | `e73c974` (merged by `093d007`) |
| Reviewed API contract | `17fd56d` |
| SQLite read model | `073561da` → integrated `a37799c` |
| HTTP, SSE and serve command | `d8ed703` |
| Frontend adapter | `56d011d` → integrated `4bebca8` |
| Reviewed reconnect corrections / combined source | `e8e022a` → integrated `a0c7e24` |

The browser used the Go implementation at `d8ed703` (binary built at `4bebca8`, with identical Go files) and the final frontend production build at `a0c7e24`, asset `index-FHJQ7uwS.js`. Subsequent documentation changes do not alter that tested source.

## Automated and independent checks

- `go test ./... -count=1` and `go vet ./...`: passed, including CLI/core/Git/setup/store and integration packages.
- Windows executable build and `GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build ./cmd/conductor`: passed. Linux runtime and a real SSH tunnel were not exercised.
- `npm test`: 6 files, 41 tests passed. `npm run build`: TypeScript and Vite passed. Vite reports a nonfatal 504.17 kB bundle size advisory (159.06 kB gzip).
- Real SQLite tests cover coherent snapshots under an independent writer, hierarchy/dependency/policy mapping, corrupt references, external content writes without events, progress events, cancellation, heartbeat, reconnect and stream limits.
- HTTP tests cover local access checks, safe errors, bodyless GET enforcement, 405 for mutations, bounded reads and shutdown draining.
- Independent read-model and transport reviews completed. CON-21 was accepted by a distinct reviewer.
- Independent frontend review found and verified corrections for terminal EventSource errors failing to retry, and pre-disconnect responses incorrectly clearing the stale warning. Both regressions failed before fixes and pass now. CON-22 was accepted by a distinct reviewer.

## Browser observations

Served the built portal and API together at `http://127.0.0.1:7331`. The explicitly isolated QA registry was `.superpowers/backend/e2e/home`; project DEMO UUID `af1d0fe7-6619-4e90-aa2f-22fa18afa82c`, plus EMPTY. This was test data only, not a case-study implementation or production ticket completion.

| Action | Observed result |
| --- | --- |
| Initial live board | Four actual database tickets; DEMO-2 assigned to `qa-worker`; DEMO-3 blocked by DEMO-2 with its readable reason. |
| Click parent chip | Epic DEMO-1 and its two children remained; unrelated DEMO-4 disappeared. |
| Switch to EMPTY, then DEMO | Empty state and zero tickets, then the correct four-ticket board; prior project filters reset. |
| External CLI claims DEMO-2 | Without reloading, the ticket moved Ready → In progress and the count changed to one. |
| Stop the test server with parent filter active | Existing cards remained, with Disconnected and an explicit stale-data warning. |
| Submit DEMO-2 through CLI while server was stopped, then restart | Without reloading or pressing retry, the browser reconnected, moved the ticket to Review, cleared the warning and retained the parent filter. |
| Accept the synthetic prerequisite through CLI | Without reloading, DEMO-2 moved to Done; DEMO-3 moved Blocked → Ready and its blocker disappeared. |

Accessibility snapshots and a visual screenshot were inspected in the browser tool. CLI mutation envelopes, fixture state, process logs and build/test output remain under `.superpowers/backend/` locally. The synthetic session was stopped and its claim released.

After the isolated checks, the final binary was opened read-only against the existing shared Conductor home. The browser loaded CON with 24 real tickets and offered CON/HOST/INT/READ project selection. Search matched real ticket keys and descriptions. No shared registry migration or installed executable replacement occurred.

## Compact-description follow-up (CON-25)

The user requested shorter cards after inspecting live data. Descriptions now collapse to four lines, with Show more / Show less only when the rendered text overflows. Full text remains in the safe highlighting/search path. Resize observation adapts the control to card width, and the toggle exposes its expanded state and controlled description. Independent code review found no issues; all 41 frontend tests and the production build passed. Browser inspection on the actual CON board at a narrow viewport confirmed truncation, expansion to full text and collapse. The user subsequently authorized merging the feature branch into main. Active-claim visibility is separately tracked in CON-26; assignment remains the current indicator.

## Run from source

Build the Go executable and `web/portal` production assets, then run:

```text
conductor --home EXISTING_DATA_HOME serve --listen 127.0.0.1:7331 --assets web/portal/dist
```

The installed shared CLI was not replaced. Use the candidate binary built from this branch until this change is landed and installed. The portal is read-only; agents continue to mutate tickets through the existing CLI. [Frontend development and fixture mode](../portal/live-integration.md) and [API details](api-contract.md) are documented separately.
