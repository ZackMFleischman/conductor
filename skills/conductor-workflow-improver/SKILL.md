---
name: conductor-workflow-improver
description: Monitor workflow reports during an authorized Conductor run, decide when retrospective analysis is useful, and feed proposed remedies into the orchestrator's shared queue.
---

# Monitor and propose improvements

Read [retrospective commands](../conductor-work/references/retrospective-commands.md) and [team commands](../conductor-work/references/team-commands.md). Restore the supplied project/run scope, identity, launch registration, problem checkpoint, deferred decisions, and Workflow Improvements epic. Acknowledge readiness with the actual session and installed skill version before reporting operational status.

Use supported native change events and due revisit triggers. If unavailable, poll only the lightweight report status at the configured interval and back off unchanged polls within the required response deadline. If no interval is configured, start at 60 seconds and back off to five minutes while unchanged; reset after a change and honor explicit deadlines. Fetch a bounded batch only when pending changes exist. Unchanged input produces no retrospective, report, checkpoint write or coordinator message. Hold no database transaction or execution claim while idle. Treat registry/authentication errors as failures, with bounded retries and a reported pause.

Invoke conductor-retrospective for new evidence or a due revisit. Classify active correctness, data, ownership, or security failures first; then blocking or recurring workflow failures; then clear user-requested efficiency or UX corrections; then lower-impact cleanup. Within a class, use arrival order unless a dependency makes another ready remedy safer. Do not defer a ready workflow remedy to an unrelated product milestone. Record the reason and a revisit trigger when evidence is insufficient or a lower-impact item is blocked.

Create or link improvement tickets under the existing meta epic; configured planning policy makes them gated drafts. Deduplicate against open remedies and preserve recurrence after delivered fixes. Send a ready remedy, its priority, evidence, and next action directly to the dedicated workflow fixer. Inform the coordinator when capacity, an active assignment, shared resources, or rollout timing needs a decision. The fixer implements one active remedy at a time; the observer remains independent and verifies later effectiveness or recurrence. Do not spawn a separate pool, become another orchestrator, or rewrite active shared skills yourself.

When evidence verifies an environment correction, format it using [validated startup corrections](../conductor-work/references/startup-corrections.md) and send the complete record to the orchestrator: source reports and evidence, project/run/host/platform/tool/operation/resource scope, exact instruction or bounded substitution, current validity, invalidation or supersession, and limitations. Forward later invalidation and recurrence evidence as well. Do not silently widen authority or treat an unverified workaround as current.

Follow remedies through independent validation and a later fresh-worker repeat of the affected workflow. Append effectiveness or recurrence evidence without erasing failures. Coordinate affected active workers to use fresh contexts after a shared-skill rollout. Reviewed shared guidance is ready for rollout without waiting for an unrelated product milestone; current workers do not automatically reload new instruction bytes.

Checkpoint on stop and end the session through the normal lifecycle. A restarted improver restores coverage and deferred triggers; it must not create duplicate remedy tickets or silently mark unseen events processed. The skill runs only while its host keeps the agent alive and does not install scheduling or hooks.

Apply [ticket comment discipline](../conductor-work/references/ticket-comments.md): use the shared call budget and minimal record formats; persist only necessary changes, never routine loops or unchanged observations.
