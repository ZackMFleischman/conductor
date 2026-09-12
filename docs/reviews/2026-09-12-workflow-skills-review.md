# Team workflow design review

Scope: the [team workflow contract](../superpowers/specs/2026-09-12-conductor-team-workflow.md), reconciled agent-deliverables/product specifications, and work board. This is design review, not runtime acceptance.

An independent reviewer examined fidelity to the original lightweight intent, preparation before execution, worker discoveries, fresh single-ticket execution, startup/recovery, adaptive concurrency, and the improvement loop.

| Finding | Resolution |
| --- | --- |
| Excluding only the implementing session does not establish independent validation. | Require a distinct validator identity, fresh context, and no implementation participation across attempts. |
| A new tracker session could still retain the previous worker's conversation. | Require a fresh host context as well as a new tracker session; reconstruct from durable records. |
| A four-slot host cannot run a separate observer, orchestrator, improver, and two workers together. | Count all applicable roles; require sufficient supported capacity to claim concurrent implementation was validated, otherwise disclose that gap. |

The reviewer rechecked these fixes and the user-requested separation of plan review, execution authority, result validation, and delivery policy. No remaining blocking issue was reported. Autonomous covered work has no compulsory human gate; automated evidence and independent review remain distinguishable.

Root consistency review removed the old persistent-worker/fixed-two-worker defaults, updated retrospective ownership and installed-versus-planned status, and linked the explicit delivery dependency graph. Markdown whitespace and local file-link checks passed. No executable, installed skill, or database schema was changed; no runtime test or case study is claimed by this review.

Next implementation must turn each dependency-graph increment into prepared tickets with concrete core/CLI contracts and behavior tests. The case study remains gated on functioning skills and explicit project/scope approval.
