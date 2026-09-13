# Portal read API v1

Implements CON-19 and CON-13 against schema 3. The frontend proposal at `feat/portal:docs/portal/api-contract.md` is retained; this document resolves its server semantics. Backend ownership is `internal/portalapi/**`, new `internal/cli/serve.go` and its tests, and `docs/backend/**`. No frontend, workflow-core, schema or installer changes.

## Transport

`conductor --home PATH serve --listen 127.0.0.1:7331 [--assets DIR]` runs in the foreground. The default is loopback only; non-loopback listen addresses are rejected. An optional trusted built-portal directory serves the application from the same origin. Development uses the frontend dev server's `/api` proxy. No CORS wildcard, authentication service, daemon, broker or write endpoints. This is a trusted local-user service; SSH forwards its loopback port for remote viewing. Untrusted local users are outside this first version's access model.

Only loopback Host names and same-origin browser requests are allowed, including forwarded localhost ports. Read failures return safe JSON errors, never database paths/SQL. Missing project: 404. Unavailable/unsupported/corrupt store: 503. Mutating methods: 405. The service opens the existing registry read-only, requires schema 3, and never creates or migrates it.

`GET /api/v1/projects` returns `{ "projects": [{"id":"UUID","name":"CON","description":""}] }`. Until display metadata exists, name is the project prefix and description is empty; repository paths are not exposed as descriptions.

`GET /api/v1/projects/{projectId}/board` returns the proposed board object directly, plus an additive opaque `revision` string. IDs are exact project UUIDs, not prefixes. Every project ticket appears once in stable creation/id order. Arrays are always arrays, including empty tickets/ancestors/blockers. Unassigned is explicit null. Text is passed through as text/Markdown source, never rendered as HTML by this API. Responses use `Cache-Control: no-store`.

## Complete ancestry and display mapping

Each ticket contains id, key, title, description, status, assignee, blockers, and `ancestors` ordered nearest parent to root; each reference has id/key/title. Compute ancestry from every ticket in the project, not just currently filtered rows. Empty means known absence. Missing metadata/parent/dependency/assignee, cross-project references, or ancestry cycles fail the whole snapshot instead of leaking another project's data or silently omitting tickets. No pagination or truncation in v1.

All mapping reads use one short SQLite read transaction. The browser never determines claim eligibility.

| Stored state/condition | Display status and reason |
| --- | --- |
| done | done; historical blockers do not change accepted work |
| review | review; additional active blockers remain visible |
| in_progress | in_progress; paused/dependency/deferral reasons remain visible |
| blocked | blocked; stored blocker reason or explicit unavailable-details fallback |
| draft | blocked; awaiting preparation/authorization |
| ready with unfinished prerequisite, active deferral, paused policy, or stale preparation/authorization | blocked; explicit reasons |
| ready otherwise | ready |

An unfinished prerequisite produces `{reason:"Waiting for KEY: title",ticketKey:"KEY"}`. Completed prerequisites do not block. Active milestone deferrals include their condition. Policy preparation/authorization compares to current specification revision. Assignment uses the agent's recorded name and identity and makes no process-liveness claim. Parent membership itself never creates an execution dependency.

## Live invalidation

`GET /api/v1/projects/{projectId}/events` is an SSE stream. On **every** connection (including reconnect), send `event: board.changed`, `id: REVISION`, `data: {"revision":"REVISION"}` immediately. The client fetches the complete board on this signal. `Last-Event-ID` is not a replay request: always refetch, so reconnect catches arbitrarily old missed changes without a retained log.

The service compares fresh coherent board snapshots about once per second while connected. The opaque hash includes displayed data and that project's durable event watermark, so CLI writes from separate processes/worktrees, note/progress mutations, and direct changes affecting board content are observed. It does not rely on in-process callbacks or watching a WAL file. Connection-first or snapshot-first startup is safe because each stream starts with unconditional invalidation. A mutation during a refetch is detected on the next comparison. The frontend must serialize/coalesce refetches or discard older responses.

Unchanged boards emit heartbeat comments every 15 seconds. Each poll ends its read transaction before waiting or writing to the socket. At most 16 streams per service, excess receives 503; disconnected clients immediately release capacity. Each SSE write/flush has a five-second deadline. A database failure emits safe `board.error` data and closes the stream; the client marks live data stale and retries, never substitutes fixtures. This is near realtime, normally within one polling interval plus query/network time.

## Acceptance and handoff

Tests use real temporary SQLite registries and independent writer connections. Cover empty/multiple projects, all lifecycle mappings, ancestor chains/cycles/missing references, consistent identities, parameterized unknown-project lookup, initial event, separate-writer change, reconnect, unchanged heartbeat, cancellation, stream limits and safe errors. CLI tests cover loopback enforcement and cancellation; run full Go tests/vet and Linux cross-build after integration.

The portal owner supplies the live adapter/runtime decoder and event-driven refetch/error handling without changing fixture behavior silently. Combined browser acceptance must demonstrate a CLI ticket mutation appearing without reload, parent subtree filtering, and reconnection, recording exact backend/frontend commits. Backend server tests alone do not establish that browser integration.
