# Retrospective commands

Contract: `conductor-team/v2`, schema 2. These commands perform bounded, persistent review. They do not run a watcher, start agents, or authorize implementation. Use the installed executable and configured `--home`; examples abbreviate that prefix as `conductor`. Every command accepts `--json` and explicit `--project PROJECT_ID`.

## Review and restart

```text
conductor retrospective status --json
conductor retrospective begin --session S --limit 50 --request review-1/begin --json
conductor retrospective show BATCH_ID --json
```

`begin` captures at most 100 problem events (default 50), including `problem.append` events on old reports. It returns `id`, exclusive `from_seq`, inclusive `through_seq`, `status`, and ordered `changes` containing event sequence, problem ID, kind, body/payload, coverage, and decision ID. One open batch is shared per project. Repeated or concurrent begins return that batch; a restarted processor reads `status.open_batch_id` and `show` to reconcile actual coverage. A replayed request can return its historical result, so use `show` before acting on it.

Do not treat failure envelopes as no work. `status.pending_changes` counts problem events above the committed watermark. Finish even an empty batch with `commit` before waiting. No transaction or ticket claim is held between commands.

## Record evidence and decisions

Create `decision.json` using IDs/sequences from the batch:

```json
{
  "batch_id": "BATCH_ID",
  "group_key": "setup-command-failure",
  "action": "milestone",
  "observation": "Two attempts failed with the same recorded command.",
  "inferred_cause": "The setup reference may be stale; this remains a hypothesis.",
  "rationale": "The workaround permits current delivery, so prepare a remedy at the release boundary.",
  "revisit_trigger": "Release validation, an active blocker, or the review deadline.",
  "review_after": "2026-10-01T17:00:00Z",
  "event_seqs": [42, 47],
  "title": "Verify and repair the setup reference",
  "body": "Reproduce the cited failures, repair the reference, and validate in a fresh session.",
  "milestone_id": "release-validation",
  "condition": "The release validation evidence has been recorded."
}
```

```text
conductor retrospective decide --session S --body-file decision.json --request review-1/decision-1 --json
conductor retrospective commit BATCH_ID --session S --request review-1/commit --json
```

Actions are `now`, `milestone`, and `observe`. Every decision requires an observation, rationale, revisit trigger, RFC3339 review deadline, stable project-scoped group key, and 1–100 unique event citations from this batch. `inferred_cause` is optional and separate from observation. Keep group keys consistent across reviews. The body is limited to 256 KiB; unknown JSON fields and conflicting session IDs are rejected.

`observe` omits all remedy and milestone fields. `now` supplies remedy title/body; `milestone` also supplies milestone ID and condition. The workflow policy must already be configured to create remedies. Retrospective creation inherits that policy and produces draft work under one persistent **Workflow Improvements** epic. Creation, decision, citations, and group links commit atomically. An existing open remedy for the group is reused. Optionally supply `ticket_id` to link a pre-existing unclaimed draft already parented under that epic; a conflicting existing group link is rejected.

Only one decision may cover a given event. If another processor won, `COVERAGE_CONFLICT` means inspect the persisted batch instead of retrying with new remedy identities. Retain each mutation's request ID and exact payload for ambiguous retries. Different events concurrently assigned to the same group still share one open remedy.

`commit` refuses incomplete coverage. It advances the persisted watermark only after every event in the captured range has a decision. Events arriving after `through_seq` remain for the next batch. Coverage does not delete events or reports. A report after a linked remedy reaches `done` creates a fresh decision with `recurrence_of`; actionable recurrence creates a new draft remedy under the same epic.

## Revisit and milestone release

```text
conductor retrospective list --limit 50 --json
conductor retrospective list --due --limit 50 --cursor ROWID --json
conductor retrospective decision DECISION_ID --json
conductor retrospective release-milestone --milestone release-validation --session COORDINATOR_SESSION --coordination TOKEN --body-file milestone-evidence.txt --request milestone/release-1 --json
```

`list` returns bounded decision summaries in row sequence order; pass the returned `next_cursor` with the same filter until omitted. `--due` selects elapsed review deadlines for the latest decision in each group. Start each new due scan without a cursor. Append review findings to the relevant problem using `problem append`, then cite that new event in the next batch to record a revised decision. A deadline requests review; it does not remove a deferral or prove the condition happened. Decision records retain rationale, citations, original milestone condition, remedy links, and recurrence links. `status` also reports the watermark, epic, pending changes, deferred count, and due count.

Milestone release requires the active project's coordinating session and token, plus evidence. An ordinary worker cannot release the gate. Release clears preparation and execution approval and returns affected tickets to draft. Normal preparation and authorization are still required; the release does not dispatch workers. Existing claims prevent release. An urgent `now` decision reuses an existing remedy but does not bypass its active milestone deferral: the coordinator reconciles that gate explicitly.

## Effectiveness

```text
conductor retrospective effectiveness DECISION_ID --session S --outcome worse --body-file repeat-evidence.txt --request review-1/effectiveness-1 --json
conductor retrospective decision DECISION_ID --json
```

Outcomes are `improved`, `unchanged`, `worse`, or `unknown`. Evidence is required and append-only. A repeated request cannot duplicate it. The decision detail includes the latest 100 observations and an explicit `effectiveness_truncated` flag; all observations remain in the event history. Keep delivery evidence and later observed effectiveness separate. A completed remedy alone does not establish improvement.
