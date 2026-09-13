---
name: conductor-workflow-improver
description: Monitor workflow reports during an authorized Conductor run, decide when retrospective analysis is useful, and feed proposed remedies into the orchestrator's shared queue.
---

# Monitor and propose improvements

Read [retrospective commands](../conductor-work/references/retrospective-commands.md) and [team commands](../conductor-work/references/team-commands.md). Restore the supplied project/run scope, identity, launch registration, problem checkpoint, deferred decisions, and Workflow Improvements epic. Acknowledge readiness with the actual session and installed skill version before reporting operational status.

Watch new reports and follow-ups through the persisted change feed, using supported native events or interruptible polling (default 30 seconds). Check for changes cheaply; do not run a full retrospective on unchanged input. Hold no database transaction or execution claim while idle. Treat registry/authentication errors as failures, with bounded retries and a reported pause.

Invoke conductor-retrospective for new evidence or a due revisit. Choose immediate preparation for an active blocker or evidenced recurring failure; a specific milestone/deadline for a useful but disruptive remedy; or observation when evidence is insufficient. Do not confuse urgency with authorization. Persist decision coverage and ticket links before advancing the checkpoint, and include follow-ups on previously reviewed reports.

Create or link improvement tickets under the existing meta epic; configured planning policy makes them gated drafts. Deduplicate against open remedies and preserve recurrence after delivered fixes. Send actionable decisions to the orchestrator and retain their substance in the tracker. The orchestrator owns shared priority, worker capacity, dispatch and any delegated preparation/authorization. Do not spawn a separate pool, become another orchestrator, or rewrite active shared skills yourself.

When evidence verifies an environment correction, format it using [validated startup corrections](../conductor-work/references/startup-corrections.md) and send the complete record to the orchestrator: source reports and evidence, project/run/host/platform/tool/operation/resource scope, exact instruction or bounded substitution, current validity, invalidation or supersession, and limitations. Forward later invalidation and recurrence evidence as well. Do not silently widen authority or treat an unverified workaround as current.

Follow remedies through independent validation and a later fresh-worker repeat of the affected workflow. Append effectiveness or recurrence evidence without erasing failures. Coordinate skill rollouts at milestones/run boundaries; current workers do not automatically reload new instruction bytes.

Checkpoint on stop and end the session through the normal lifecycle. A restarted improver restores coverage and deferred triggers; it must not create duplicate remedy tickets or silently mark unseen events processed. The skill runs only while its host keeps the agent alive and does not install scheduling or hooks.
