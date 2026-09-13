# Native-host team commands (contract 2)

These commands record coordination and host observations. Use native host tools
to create, query, checkpoint and stop children. Conductor launches no process and
never treats a session timestamp as process liveness. Do not run an unverified
candidate against an existing registry.

Every mutation requires a caller-retained unique `--request KEY`. Retry a lost
response using the identical request and payload; a replay is historical, so
`team show RUN` must reconcile current state before more work. Each command emits
the standard JSON envelope. `--json` is accepted by the top-level CLI.

Start with a live coordinator session from `session start`. `team start --session
S --body-file profile.json --request KEY` creates a new immutable profile and
exclusive coordination token. Only one non-stopped run may exist per project.
The profile records the existing user grant; arbitrary ticket text is no grant.

```json
{
  "profile": {
    "scope": "Implement the approved feature within its recorded non-goals",
    "approval_reference": "User-approved proposal and revision reference",
    "skill_version": "2",
    "host": "native host name",
    "host_limit": 4,
    "child_limit": 3,
    "required_roles": ["improver"],
    "improvement_enabled": true,
    "capabilities": ["spawn", "status"],
    "polling_seconds": 30,
    "delivery_policy": "Tested worktree commits; integration separately authorized",
    "stop_conditions": "Scoped acceptance complete or explicit stop"
  }
}
```

The host limit includes the coordinator. Active, starting and unknown launches
consume child slots. Terminal host observations reclaim capacity only after
registered child sessions have stopped and their execution claims are released.
Required roles must acknowledge readiness before implementation dispatch. An
improvement-enabled run requires the improver. Unknown capabilities or host state
must be reported; do not fabricate observations to pass readiness.

All subsequent coordinator writes have this common form, using values returned by
the most recent write or `team show`:

```text
team ACTION RUN --session S --coordination TOKEN --expect-revision N --request KEY [--body-file body.json]
```

| Action | Body and behavior |
| --- | --- |
| `resume` | No body. Reconciles the same held coordinator and advances readiness epoch; it does not assume children are alive. |
| `launch` | `{"role":"improver","agent_id":"AGENT_UUID"}`. Records a unique intent **before** native host creation. Roles: improver, planner, reviewer, validator, worker, observer. Worker requires `ticket_id` and a ready run; use the exact ticket UUID. |
| `register` | `{"launch_id":"L","host_id":"HOST_ID","child_session_id":"CHILD_SESSION"}`. Binds the intent to one fresh native context and session matching its stable identity. Neither host ID nor session can be reused by another launch. Identical registration is safe; different registration is rejected. |
| `observe` | `{"launch_id":"L","host_state":"active","evidence":"Native tool result or checkpoint reference"}`. Host states: active, unknown, finished, stopped. Host transitions advance epoch and invalidate old acknowledgements. Unknown launch results must be reconciled before retrying creation. |
| `ready` | No body. Requires capabilities, live coordinator and child sessions, current native active observations and fresh acknowledgements from required roles. |
| `stop` | `{"reason":"Checkpoint requested; native child statuses inspected"}`. Stops dispatch and records stopping while any child remains unresolved. After native stops, claim release, child session stop and terminal observations, repeat with a new request/revision to finish the run and release coordination. |
| `release` | `{"reason":"Durable checkpoint and unresolved child status"}`. Releases only coordination; preserves all child records, sessions, execution claims and scope. A released run requires explicit recovery before reuse. |

Registration is bookkeeping after the native create result. Pass the launch ID,
run ID, stable identity, scope, skill version and exact assignment to the fresh
child. The child starts a new tracker session, then is registered and responds to
the current challenge using its own session:

```text
team ack RUN --session CHILD_SESSION --expect-revision N --body-file ack.json --request KEY
```

```json
{
  "launch_id": "L",
  "epoch": 2,
  "host_id": "HOST_ID",
  "agent_id": "AGENT_UUID",
  "role": "improver",
  "scope": "Exact immutable profile scope",
  "skill_version": "2",
  "checkpoint": "Current exact assignment or report checkpoint"
}
```

Each successful write increments run revision, including child acknowledgements.
After a host transition, observe unchanged active children again and request
their new acknowledgements at the current epoch. A stopped tracker session cannot
acknowledge or resume. Readiness is computed on inspection as well as mutation;
`status: ready` alone is insufficient, and the `ready` boolean and
`readiness_issues` are authoritative for the recorded observations.

Explicit recovery requires inspecting the former supervisor, children and
uncommitted files first. It rotates the coordinator fence and readiness epoch,
preserves execution claims, and never kills or takes over a process. It uses a
distinct fresh replacement session with no previous coordination, managed launch
or execution claim, and must be a real authorized human recovery action:

```text
team recover RUN --session NEW_SESSION --expect-revision N --human --body-file recovery.json --request KEY
```

`recovery.json` contains `{"reason":"Inspection and authorization evidence"}`.
Do not issue `--human` as an agent to bypass recovery authority. A stopped run
cannot revive; start a new run under the same stable coordinator identity using
a new tracker session. Every coordinator mutation is fenced; obsolete tokens
cannot mutate after release or recovery.

Inspection uses the read-only store path and starts no session:

```text
team show RUN
team list
team events RUN --after 0
team --help
```

`events` returns up to 100 ascending events with actor, input, revision and epoch.
Continue with `--after` set to the last returned `seq`. `list` returns the most
recent 100 runs. Project/session/launch lookups are project-scoped. Coordination
is distinct from ticket ownership: teams never release ticket claims implicitly.
