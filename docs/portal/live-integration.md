# Live portal integration

The portal opens in live mode. It loads `/api/v1/projects`, selects the first registered project, and lets the viewer change projects using the Project selector. An empty registry is distinct from an empty board. Failed requests show safe errors and never substitute fictional tickets.

## Run

Build the frontend using the locked dependencies in `web/portal`:

```powershell
npm ci
npm test
npm run build
```

From the repository root, run the backend built from the integrated source:

```text
conductor --home PATH serve --listen 127.0.0.1:7331 --assets web/portal/dist
```

Open the service's loopback address. Production uses same-origin URLs. For frontend development, run the service on port 7331 and `npm run dev` in `web/portal`; Vite proxies `/api` to that loopback service. Do not use Vite's static preview for live acceptance: it has no backend proxy. Use `?mode=fixture` explicitly for the fictional demo, which remains labeled “Fixture data.” Remove that query parameter to return to live mode.

## Runtime behavior

`liveBoardSource.ts` decodes unknown JSON before display. It checks required strings and arrays, allowed statuses, explicit null assignment, unique ticket IDs/keys and project IDs, assignment identity consistency, blocker keys, and exact reference identity/title/key agreement. Each nearest-parent-to-root chain must match its parent's chain and contain no cycles. API v1 returns every project ticket, so an ancestor missing from the snapshot is rejected. This live decoder does not change the separate fixture adapter's ability to demonstrate ancestors absent as rows.

The opaque revision is validated but never compared numerically. Every `board.changed` event, including the initial event and reconnect events, requests a snapshot. One request runs at a time; multiple events during that request become one follow-up request, and the superseded result is discarded. Source changes and unmount abort pending loads, remove listeners, close the EventSource, and cancel scheduled reconnects. Project changes reset filters, avoiding invisible filters left over from another project. Refreshes within a project retain filters.

Network disconnects use EventSource reconnection while the stream is reconnecting. A terminal native error (for example an HTTP 503), `board.error`, or malformed invalidation closes that stream and schedules a fresh connection after three seconds. The last successful board remains visible with a stale warning through reconnection until a fresh snapshot succeeds. A failed refresh also preserves the last successful board. “Try again” restarts the load and connection immediately. No server error body or exception details are shown.

## Validation handoff

Unit/component tests exercise decoder failures, live first load, invalidation and reconnect, serialized refreshes, discarded obsolete responses, safe failures/retry, project switching, abort/listener cleanup, timed stream recovery and timer cleanup. The existing fixture, plain-text rendering, ancestry and combined filter tests continue to run.

Combined browser acceptance belongs to the integration coordinator. Use an isolated registry and the integrated backend/frontend source: create or update a ticket through a separate CLI process and observe the board update without a reload; check parent-subtree plus owner/search filtering; stop/restart the service and verify a stale warning followed by fresh content; switch projects during a request; inspect desktop and narrow screens. Record exact source commits and the final integration commit. Component tests alone do not establish real-service browser behavior.
