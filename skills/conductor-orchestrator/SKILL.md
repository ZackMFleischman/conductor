---
name: conductor-orchestrator
description: Start or resume an authorized Conductor team run, prepare and review its work, reconcile managed agents, and dispatch fresh workers within actual capacity.
---

# Coordinate a run

Ordinary repository work does not invoke the team lifecycle. Use this skill for an explicit orchestration request or a known authorized run resume. Check registration and read [team commands](../conductor-work/references/team-commands.md) and [workflow commands](../conductor-work/references/workflow-commands.md). Use the installed executable/data home; never create a fallback registry on failure.

## Every start or resume

1. Check compatible CLI/skills, store access, actual native host creation/status/message/wait capabilities, and user-granted scope. Load the run profile, approval state, roles, identities, resource limits, and stop conditions. Validate reduced-operation policy explicitly if the host cannot manage agents.
2. Acquire/reconcile the exclusive project coordination fence. Restore launch intents, child host IDs, claims/worktrees, and checkpoints. Quiet or old contact does not establish death; recover ownership only after inspecting the old coordinator and its children.
3. Inspect actual host status and obtain fresh readiness acknowledgements from required roles. An identity row or session write is not evidence of a live process. In an improvement-enabled run, ensure conductor-workflow-improver is running and has restored its report checkpoint. Never say the team is ready while a required role is absent or unknown.
4. Reuse healthy children. Before a new child, persist a launch intent with its scope and unique ID, then invoke supported native host creation with that ID. Attach the returned host ID and require child registration/acknowledgement. Reconcile an ambiguous launch before retry; unknown starts consume capacity. Never hold a database transaction across host operations.

Use conductor-plan before implementation dispatch. Team coordination and planning policy are independent opt-ins: read the actual ticket policy, and do not require or enable gated drafts merely because a team exists. Existing policy may delegate plan review and execution authorization to agents; human involvement is only required for retained decisions or out-of-scope actions. Discoveries and improvement remedies enter the same prioritization queue, with preparation gates where configured. Track material plan changes and affected work; do not silently approve a different scope.

## Dispatch and follow-through

Choose a supported model and effort for each fresh task. Start with a light model for routine observation, a bounded coding/review model for focused changes, and escalate after complexity, failed attempts, or unresolved risk warrants it. Record the choice and short reason in the handoff. Use focused contexts. Choose concurrency from independent ready work, host limits, and resources. Count yourself where the host counts parents, plus the improver, reviewers, validators, starting/unknown children, and workers. Consider file/interface overlap, ports, test data, memory, and browser load. Do not hard-code a smaller project limit or fill slots with idle workers when no useful ticket is ready.

Start a fresh host context with conductor-worker for one exact ticket, stable identity, project, scope, approved revision/context links, safe worktree, and run/launch IDs. Read [validated startup corrections](../conductor-work/references/startup-corrections.md) once per relevant context; before each launch, compare verified-current records with the actual project, run, host, platform, tool, operation and resource, and embed each applicable concrete instruction plus source IDs under `Applicable validated corrections`. Resolve `CURRENT_CHECKOUT` to the exact worker checkout, but do not invent a subcommand that the assignment did not authorize. Exclude mismatched or invalid records; a reference link alone is not consumption. Workers validate applicability before their first affected command and claim their own assignments. Allow ticket creation, not scope expansion or recursive worker pools. Reuse stable identities only after prior attempts are reconciled. A submitted ticket does not free a slot until the host confirms its worker finished.

Allocate independent review/validation capacity, not just implementation capacity. Persist only assignment-changing decisions not already recorded by a transition or linked handoff; use the shared three-bullet record format. Group optional PRs by coherent delivery; integrate serially and validate the exact combined target SHA. Refresh evidence after the target changes. Do not equate an empty queue with completed goals.

The improver proposes urgency and remedies; the dedicated workflow fixer takes one ready remedy at a time in priority order. You retain capacity, resource, and conflicting-integration decisions. Shared skill changes take effect through the normal reviewed installation path. Coordinate affected agents explicitly to start fresh contexts, then ship the reviewed result as ready; do not gate it on an unrelated product milestone.

When idle use native event-driven, interruptible waits without a claim or transaction held merely to poll. An unchanged timeout triggers no Conductor read, observation write, acknowledgement or checkpoint. Reconcile only an actual host/assignment change, recovery, required fence transition or due decision. If native change notifications are unavailable, use the run's configured polling interval and back off unchanged polls within its response deadline; read only the needed state. If no interval is configured, start at 60 seconds and back off to five minutes while unchanged; reset after a change and honor explicit deadlines. Bound transient retries; storage or authentication failure is not an empty queue. Persist decisions/checkpoints so resumption does not depend on this conversation.

On stop, cease dispatch, request checkpoints, preserve unresolved children/claims and report those still active. Use authorized host controls only. A stopped orchestrator does not prove children stopped, and this skill cannot wake a closed host. Do not add a daemon, hook, or custom launcher to simulate missing capabilities.

Apply [ticket comment discipline](../conductor-work/references/ticket-comments.md): use the shared call budget and minimal record formats; persist only necessary changes, never routine loops or unchanged observations.
