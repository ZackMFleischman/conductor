---
name: conductor-worker
description: Execute one assigned Conductor ticket in a fresh agent context, record discoveries and evidence, and end the worker session after its handoff.
---

# One assignment

Use conductor-work and its [commands](../conductor-work/references/commands.md). Read [workflow commands](../conductor-work/references/workflow-commands.md) for readiness, blockers, and validation. If managed by an orchestrator, read [team commands](../conductor-work/references/team-commands.md) for child registration and acknowledgement.

Start with the exact project, ticket, stable identity, run/launch IDs where applicable, and authorized scope supplied by the coordinator. This must be a fresh host conversation and Conductor session, not a new database session inside an old implementation conversation. Reconstruct context from the approved ticket, dependencies, and linked artifacts. Do not inherit an entire previous worker transcript.

Read [validated startup corrections](../conductor-work/references/startup-corrections.md) when the assignment includes them. Before the first affected command, verify each supplied record matches the actual checkout, host, platform, tool, operation and resource, then use its concrete instruction on that first attempt. Preserve source IDs and the actual invocation in evidence. Report mismatched or stale records instead of broadening them; corrections do not add authority.

Enter a safe isolated worktree before claiming. Register the actual host/session relationship and acknowledge readiness when managed. Assignment is not ownership. Claim the exact eligible ticket; do not alter assignment or policy to make a rejected claim succeed. Recover only the database-confirmed claim and recorded checkout on interruption. A stopped session requires a new session.

Execute through conductor-work. Report workflow problems when they occur. Create linked tickets for missing prerequisites, discovered bugs, and out-of-scope follow-ups after checking duplicates; planning policy determines whether they start as gated drafts. Give each enough context for a different fresh worker. Small corrections within this assignment's criteria stay in the current ticket. Creating a discovery does not authorize doing it or spawning a child.

Keep one ticket focused on one user-visible feature or bug. Record review, validation, landing, and demonstration as checklist items or status on that ticket. Do not create a ticket only for a workflow stage. Keep handoffs concise and link detailed technical evidence. Send coordination only when it changes the recipient's work or a shared resource.

If blocked, save evidence and a useful handoff, atomically block/release, and end. If release is uncertain, pause and reconcile before ending. If a specification changes materially, checkpoint and wait for the coordinator to resolve the invalidated approval; ownership is not permission to submit against superseded criteria.

Before handoff, fill the planned evidence contract criterion by criterion: required setup/preconditions, execution, actual result, exact checkout/commit, executor, and raw artifact references. Verify that each cited artifact contains the exact fact attributed to it; use separate citations for facts proven by different artifacts. If a required precondition is missing or differs from the criterion, report that criterion as incomplete and leave it out of any complete-acceptance claim even when commands passed. Submit the source commit, completed criteria, validation evidence, remaining QA, and integration/PR handoff. Respect the recorded validation policy. Independent-agent acceptance requires a distinct validator with no implementation participation; never impersonate human approval. Submission does not mean merged, integrated, or accepted.

After handoff, stop the tracker session and return a final result to the host so this worker can terminate. Do not claim another ticket. Report the host ID and any unresolved ownership to the coordinator. The orchestrator, not this worker, starts the next fresh context.

Apply [ticket comment discipline](../conductor-work/references/ticket-comments.md): retain decisions, questions, blockers, corrections and evidence; omit routine/no-op narration and read relevant history on demand.
