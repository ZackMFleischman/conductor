# Portal read API v1

Implements CON-19 and CON-13 against schema 3. The frontend proposal at `feat/portal:docs/portal/api-contract.md` is retained; this document resolves its server semantics. Backend ownership is `internal/portalapi/**`, new `internal/cli/serve.go` and its tests, and `docs/backend/**`. No frontend, workflow-core, schema or installer changes.

## Transport

`conductor --home PATH serve --listen 127.0.0.1:7331 [--assets DIR]` runs in the foreground. The default is loopback only; non-loopback listen addresses are rejected. An optional trusted built-portal directory serves the application from the same origin. Development uses the frontend dev server's `/api` proxy. No CORS wildcard, authentication service, daemon, broker or write endpoints. This is a trusted local-user service; SSH forwards its loopback port for remote viewing. Untrusted local users are outside this first version's access model.

Only loopback Host names and same-origin browser requests are allowed, including forwarded localhost ports. Read failures return safe JSON errors, never database paths/SQL. Missing project: 404. Unavailable/unsupported/corrupt store: 503. Mutating methods: 405. The service opens the existing registry read-only, requires schema 3, and never creates or migrates it.

GET requests must be bodyless; a declared body receives 400 and connection close without draining it. Mutating methods remain 405 even with a body. Shutdown cancels streams and waits for HTTP connections to drain before closing the database and assets.

`GET /api/v1/projects` returns `{ "projects": [{"id":"UUID","name":"CON","description":""}] }`. Until display metadata exists, name is the project prefix and description is empty; repository paths are not exposed as descriptions.

`GET /api/v1/projects/{projectId}/board` returns the proposed board object directly, plus an additive opaque `revision` string. IDs are exact project UUIDs, not prefixes. Every project ticket appears once in stable creation/id order. Arrays are always arrays, including empty tickets/ancestors/blockers. Unassigned is explicit null. Text is passed through as text/Markdown source, never rendered as HTML by this API. Responses use `Cache-Control: no-store`.

## Complete ancestry and display mapping

Each ticket contains id, key, title, description, status, assignee, blockers, and `ancestors` ordered nearest parent to root; each reference has id/key/title. Compute ancestry from every ticket in the project, not just currently filtered rows. Empty means known absence. Missing metadata/parent/dependency/assignee, cross-project references, or ancestry cycles fail the whole snapshot instead of leaking another project's data or silently omitting tickets. No pagination or truncation in v1.

Ticket details also include additive string fields `kind` (the recorded work type), `summary`, `evidence`, `qa`, `createdAt`, and `updatedAt`. The text fields retain their Markdown source; timestamps retain their stored values. Older snapshots without these fields remain readable by the frontend. Done pagination is presentation-only: the full snapshot remains available for search and ticket links. The activity sidebar compares accepted snapshots in memory, keeps the latest 200 ticket changes since opening the board, and may combine intermediate changes during reconnect; it is not an audit log.

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

## Workflow reports and ticket comments

Each ticket includes `commentCount`, the count of explicit `ticket.note` events scoped to that project and ticket. Cards show positive counts and the collapsed Comments control shows the count without fetching history. Counts refresh with the board snapshot; comment bodies remain on-demand. Older snapshots may omit the field.

The board includes lightweight `problems` summaries with `id`, `key`, `summary`, optional related `ticketKey`, `createdAt`, `updatedAt`, and `noteCount`, newest reported first. Reports appear in a separate Problems tab; they do not introduce a ticket state or an inferred open/resolved status. Search covers the summary, key and related ticket. Report changes participate in board revision and the session activity feed.

`GET /api/v1/projects/{project}/problems/{problem}` loads a report's expected behavior, actual behavior, correction, evidence, reporter and chronological append history on demand. Problem and ticket references open their corresponding detail dialogs. Unknown or cross-project report IDs return 404.

`GET /api/v1/projects/{project}/tickets/{ticket}/notes?before=CURSOR` returns `{ticketId, notes, total, nextCursor?}`. Notes contain `id`, `author`, `body` and `createdAt`; only explicit ticket-note events are included, newest first in pages of 20. The opaque cursor preserves the event-sequence boundary. Invalid cursors return 400 and missing/cross-project tickets return 404. The Comments section starts collapsed, fetches only when opened, and refreshes the recent page when the open ticket changes. Older comments load explicitly. These routes are read-only and retain the same project scoping and local-service access model.

Done tickets additionally expose `completedAt`, derived from the latest `ticket.accept` event, with `updatedAt` as a fallback for legacy records. The UI sorts Done by this timestamp descending before applying its 20-card limit. Later comments do not reorder completed tickets. Other columns retain their existing order.

## Live invalidation protocol

`GET /api/v1/projects/{projectId}/events` is an SSE stream. On **every** connection (including reconnect), send `event: board.changed`, `id: REVISION`, `data: {"revision":"REVISION"}` immediately. The client fetches the complete board on this signal. `Last-Event-ID` is not a replay request: always refetch, so reconnect catches arbitrarily old missed changes without a retained log.

The service compares fresh coherent board snapshots about once per second while connected. The opaque hash includes displayed data and that project's durable event watermark, so CLI writes from separate processes/worktrees, note/progress mutations, and direct changes affecting board content are observed. It does not rely on in-process callbacks or watching a WAL file. Connection-first or snapshot-first startup is safe because each stream starts with unconditional invalidation. A mutation during a refetch is detected on the next comparison. The frontend must serialize/coalesce refetches or discard older responses.

Unchanged boards emit heartbeat comments every 15 seconds. Each poll ends its read transaction before waiting or writing to the socket. At most 16 streams per service, excess receives 503; disconnected clients immediately release capacity. Each SSE write/flush has a five-second deadline. A database failure emits safe `board.error` data and closes the stream; the client marks live data stale and retries, never substitutes fixtures. This is near realtime, normally within one polling interval plus query/network time.

## Acceptance and handoff

Tests use real temporary SQLite registries and independent writer connections. Cover empty/multiple projects, all lifecycle mappings, ancestor chains/cycles/missing references, consistent identities, parameterized unknown-project lookup, initial event, separate-writer change, reconnect, unchanged heartbeat, cancellation, stream limits and safe errors. CLI tests cover loopback enforcement and cancellation; run full Go tests/vet and Linux cross-build after integration.

The portal owner supplies the live adapter/runtime decoder and event-driven refetch/error handling without changing fixture behavior silently. Combined browser acceptance must demonstrate a CLI ticket mutation appearing without reload, parent subtree filtering, and reconnection, recording exact backend/frontend commits. Backend server tests alone do not establish that browser integration.

## Ticket attachments and detail presentation

The board adds `attachments` per ticket (empty array when none): `id`, `name`, `url`, `mediaType`, `size` in bytes, and `modifiedAt`. These are live references, not uploaded or managed copies. The portal recognizes Markdown file destinations (including reference definitions), backtick file paths, and bare absolute file paths in description, summary, evidence and QA. Relative paths resolve against the registered repository root (parent of a standard `.git` common directory). Files must exist inside that root; external worktrees, bare repositories, missing files, Git internals and paths escaping through symlinks are not served. At most 200 candidate references and 50 existing attachments are examined/returned per ticket. No arbitrary path is accepted from HTTP clients.

`GET /api/v1/projects/{project}/tickets/{ticket}/attachments/{attachment}` rechecks the current ticket reference and root containment before streaming a regular file. Raster images render inline; other types download with their file name, nosniff and sandbox CSP headers. Missing/removed references return404. Local file size/mtime changes affect the board revision and appear through normal polling. Files remain live and may change after a snapshot; metadata is not a content hash or immutable artifact guarantee.

The frontend also collects safe HTTP(S) Markdown images and recognizable file links. Cards show the first image and attachment count. Details list all attachments with compact previews/file sizes; images open a keyboard-dismissible viewer with fit/actual-size and original-file controls. Broken images show a fallback. Raw HTML remains disabled. There is no upload or attachment-edit workflow in this increment.

The details dialog uses a readable content column plus ownership, parent/child links and dates. Delivery summary stays visible; evidence and QA can be expanded. The activity feed keeps the latest200 changes for the current board session. Archive all hides current entries; new entries stay visible, Show archived restores visibility, and project changes/resetting the page clear the session archive. Collapsing the drawer restores the header Activity button and keyboard focus.
