# Proposed portal board API contract (v1)

Status: frontend proposal, **not an installed or implemented API**. Integration request CON-13 is assigned to coordinator-1 for CON-4. Logical parent CON-8; frontend implementation CON-11 and CON-12. Core must review this proposal before either side claims live compatibility.

## Small read boundary

The portal depends on `BoardSource` in `web/portal/src/board.ts`, not CLI output, database tables, claims, or storage lifecycle enums. `load(signal?: AbortSignal): Promise<BoardSnapshot>` returns one complete project snapshot. The active source advertises `kind: 'fixture' | 'live'` independently of loading success. This increment always selects `fixtureBoardSource`; it never silently falls back from a live failure to sample data.

Suggested future transport: `GET /api/v1/projects/{opaqueProjectId}/board`, returning the JSON object below with HTTP 200. The route is a proposal only. Authentication, authorization, hosting/origin and project selection must be agreed with CON-4 before enabling a network adapter. No server or fetch adapter exists in this increment.

```json
{
  "project": {
    "id": "opaque-project-id",
    "name": "Conductor",
    "description": "Project purpose in plain text"
  },
  "tickets": [{
    "id": "opaque-ticket-id",
    "key": "CON-42",
    "title": "Make work easy to inspect",
    "description": "Complete human-readable description, with line breaks preserved.",
    "status": "blocked",
    "assignee": {"id": "opaque-agent-id", "name": "portal-coordinator-1"},
    "blockers": [{"reason": "Waiting for an agreed API contract", "ticketKey": "CON-13"}]
  }]
}
```

## Semantics and validation

| Field | Required meaning |
| --- | --- |
| project.id / ticket.id | Nonempty opaque stable identity. Ticket IDs must be unique within a snapshot. Never infer file paths or database keys. |
| project.name / description | Display name and plain-text context. |
| key / title / description | Human-facing key, short title, full plain text. Empty description displays “No description provided.” Strings are rendered as text, never HTML. |
| status | Exactly `ready`, `in_progress`, `blocked`, `review`, or `done`. Presentation status supplied by core; frontend does not infer claim eligibility or accepted state from it. |
| assignee | Object with stable id and display name, or explicit `null` for unassigned. Assignment is not an active claim, process liveness, or permission to work. |
| blockers | Required array; empty means no reported blockers. Each item has a nonempty human-readable reason and optional related ticket key. Blocked tickets should include at least one reason; UI still shows “Blocker details unavailable” if none are supplied. Blocker details may also accompany other statuses without changing their lane. |

Core owns mapping from foundation/workflow/team layers to these display states. In particular, define how draft, deferred, paused, submitted and accepted records map before implementation; do not copy storage enums into the browser. Include every ticket intended for the board exactly once, in stable source order. No hidden pagination: if the response needs pagination, negotiate it as a contract change before shipping. Empty `tickets` is a valid empty board.

A future adapter must validate unknown JSON, reject missing/invalid required data, duplicate IDs and unsupported status values, and then map into the frontend types. Unknown additive fields may be ignored. TypeScript types alone are not runtime validation. Unsupported records must not silently disappear. Transport errors and authorization failures reject loading and produce an error/retry state, distinct from a valid empty board. Cancellation uses AbortSignal. This increment's trusted in-repository fixture needs no network decoder.

No polling, write endpoint, claim action, database migration, CLI replacement or API implementation is requested here. Error details in the UI must be safe user-facing text, not raw server responses.

## Later extensions

Stable ticket IDs allow a separate detail resource and selected-ticket state. Markdown body content and linked-artifact metadata/rendering require separate contracts, trust rules and acceptance criteria. They are deliberately absent from v1; do not preload artifact contents or introduce speculative fields now.

## Integration acceptance (future work)

CON-4 / CON-13 should record the agreed route/schema and lifecycle mapping, exact landed core and portal commits, a real read adapter, response-validation tests, access-control/error cases and a browser run against the actual service. The present fixture tests and screenshots demonstrate frontend behavior only. Human QA and integration acceptance remain explicit separate steps.
