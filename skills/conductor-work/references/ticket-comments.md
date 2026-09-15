# Minimal records and bounded reads

A durable record contains only information needed to decide, resume or validate the work. Ordinary work loops and unchanged waits produce no record.

- Ticket description: outcome, scope/constraints, testable acceptance criteria, necessary dependency links. Default to 150 words or fewer; link an existing detailed specification instead of copying it.
- Note/checkpoint: at most three short bullets, normally 80 words total: changed fact or decision and reason; exact evidence link; required next action or unresolved blocker. Omit empty slots. Write only if another worker needs it before final handoff, a durable claim needs correction, or interruption requires resumption context.
- Submission: result, criterion/check outcomes at the tested source, remaining QA and next owner/action. Store each fact once in its appropriate summary/evidence/QA field. Include required provenance; link detailed logs rather than copying them.
- Problem: brief expected/actual behavior, impact or next action, supporting evidence. Report blockers, recurring defects, correctness/ownership risks or reusable fixes. Corrected typos, expected red tests and routine retries are not problem reports.

Word limits are defaults; required acceptance evidence and concrete reproducible blockers take priority. Use one canonical location and reference its ID elsewhere. Lifecycle events already record claims and transitions. A submission replaces a separate completion note. Do not copy chat transcripts, command inventories, repeated plans, IDs already in structured metadata, or unchanged status into records. Keep working notes locally; create a linked artifact only for evidence someone will need.

Preserve history: append a concise correction to a material false claim. Resolve conflicts with approved fields through authorized update/review; comments never grant scope, ownership, state changes or approval. A necessary problem report belongs in `problem add`/`problem append`; a ticket needs only its link when relevant.

Read the supplied ticket and necessary context once; retain IDs and revisions from responses. Fetch again only for missing information needed now, a signaled external change, conflict, or recovery. Do not refresh after each note or replay full history. Read older comments only to resolve a specific decision. Keep history cursors locally until a real checkpoint is needed.

`ticket show ID --json` includes at most 20 recent events and `events_truncated`; it is not a complete comments log. Do not invent `ticket comments` or cursor flags. The portal loads 20 note entries at a time with Load older on demand. If a necessary decision is missing, follow its cited artifact or a supported history read; an absent older comment is not proof no decision exists.

Example: "Blocked: fixture lacks X, so criterion Y cannot run. Evidence: [artifact]. Owner Z must supply X."
