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

Use conductor-plan before implementation dispatch. Existing policy may delegate plan review and execution authorization to agents; human involvement is only required for retained decisions or out-of-scope actions. Draft discoveries from workers and remedies from the improver enter the same preparation queue. Track material plan changes and affected work; do not silently approve a different scope.

## Dispatch and follow-through

Choose concurrency from independent ready work, host limits, and resources. Count yourself where the host counts parents, plus the improver, reviewers, validators, starting/unknown children, and workers. Consider file/interface overlap, ports, test data, memory, and browser load. Do not fill slots with idle workers when no useful ticket is ready.

Start a fresh host context with conductor-worker for one exact ticket, stable identity, project, scope, approved revision/context links, safe worktree, and run/launch IDs. Workers claim their own assignments. Allow ticket creation, not scope expansion or recursive worker pools. Reuse stable identities only after prior attempts are reconciled. A submitted ticket does not free a slot until the host confirms its worker finished.

Allocate independent review/validation capacity, not just implementation capacity. Record substantive messages in tickets. Group optional PRs by coherent delivery; integrate serially and validate the exact combined target SHA. Refresh evidence after the target changes. Do not equate an empty queue with completed goals.

The improver proposes urgency and remedies; you own shared prioritization and dispatch. Default to at most one active improvement implementation ticket within the same worker pool. Protect delivery capacity. Shared skill changes normally take effect at a milestone/run boundary, with fresh contexts and recorded versions; urgent changes require coordinated checkpoints.

When idle use native interruptible waits (default 30 seconds), without a claim or transaction held merely to poll. Bound transient retries; storage or authentication failure is not an empty queue. Persist decisions/checkpoints so resumption does not depend on this conversation.

On stop, cease dispatch, request checkpoints, preserve unresolved children/claims and report those still active. Use authorized host controls only. A stopped orchestrator does not prove children stopped, and this skill cannot wake a closed host. Do not add a daemon, hook, or custom launcher to simulate missing capabilities.
