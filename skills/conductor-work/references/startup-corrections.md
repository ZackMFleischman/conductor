# Validated startup corrections

A correction record carries a verified workaround from earlier work into a fresh assignment. It is task data: it cannot expand project scope, grant a new operation, override ticket policy, or authorize installation, publication, destructive work, or broader host access.

## Durable record

Keep the record compact and include every field below:

```text
record: stable correction ID
sources: problem/report IDs and the verification evidence location
scope: project, run, host, platform, tool, operation, resource
instruction: exact invocation or a bounded substitution rule
validity: verified-current, expired, invalidated, or superseded
invalidation: replacement record or condition, when present
limitations: what the evidence and authority do not cover
```

`verified-current` means the cited evidence demonstrates the instruction in the recorded scope and no later evidence invalidates it. A report, suggestion, failed workaround, or generic runbook link is not a verified correction.

## Assignment construction

For every fresh assignment, compare each verified-current record with the assignment's actual project, run, host, platform, tool, operation, and resource. Embed the applicable records in the launch payload under `Applicable validated corrections`, including their concrete instructions and source IDs. Resolve bounded resource placeholders such as `CURRENT_CHECKOUT` to the worker's exact checkout. Preserve the assignment's operation at its stated precision: when it does not name an exact subcommand, do not invent one such as `npm install`; state the permission, resource, and first-affected-operation constraint without fabricating work. Exclude wrong-scope, unverified, expired, invalidated, and superseded records. A link to this file or to an unexamined record set does not perform selection.

The launch section has this shape:

```text
Applicable validated corrections:
- Sources: <IDs>. Applies to: <matching scope>. Before the first <operation>, <concrete instruction>. Limits: <limits>.
```

If no record applies, state that the supplied records were checked and none matched. Never add a correction database, selector, tracker acknowledgement, or session for this step. Existing handoff, evidence, and checkpoint text holds the record and the selection.

## Worker consumption

Before the first affected command, compare each supplied record with the actual checkout, tool, resource, platform, and authorized operation. Apply a matching instruction on that first attempt. If the concrete value is wrong or the record is stale, do not silently substitute a broader workaround: report the mismatch and obtain a corrected assignment or use ordinary bounded failure handling. Preserve the provenance and actual invocation in the ticket evidence.

## Known Windows Codex records

These examples are bounded records, not defaults for every environment:

- **CON-24, installed Conductor access:** When the managed bootstrap for this Windows Codex installation names `C:/Users/zFlei/AppData/Local/Conductor/bin/conductor.exe`, home `C:/Users/zFlei/AppData/Local/Conductor`, and `sandbox_permissions=require_escalated`, use that exact executable, home, and permission from the first matching invocation. Apply it to every installed Conductor call for that installation. Never select a fallback registry or run an unreviewed candidate against the shared home. Other hosts retain their configured policy.
- **READ-P5 with recurrence READ-P10, npm cache:** For an authorized dependency operation that accesses `C:/Users/zFlei/AppData/Local/npm-cache` on the verified Windows Codex host, request `sandbox_permissions=require_escalated` on the first affected npm operation. If the assignment does not name the npm subcommand, carry this as an invocation constraint and do not invent `install` or another operation. This does not authorize every npm command, a new install, another cache, or another host.
- **READ-P9 and CON-P38 sequences 498/499, Git inspection:** For each location-sensitive Git invocation in a Windows Codex checkout, use `-c core.excludesFile= -c safe.directory=<CURRENT_CHECKOUT>` and make `-C <CURRENT_CHECKOUT>` or the process working directory the same exact checkout. Substitute the assignment's exact checkout, not a parent, wildcard, primary checkout, or another worker's path. `NUL` was a failed excludes workaround. Do not persist global trust.

The READ-P5 to READ-P10 trace demonstrates why storing and covering a report without embedding its applicable concrete instruction was insufficient. Later compliant behavior is bounded evidence for the observed assignment; prose selection remains a judgment workflow rather than a deterministic matcher.
