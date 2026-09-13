# Existing-project onboarding skill verification

Scope: CON-29, 13 September 2026 UTC. Adds a bundled onboarding skill and installer recognition using CLI 0.2/schema 3/`layers-v1`. This is skill/package validation and a read-only Lux assessment, not a completed Lux import or hardware acceptance run.

## Packaging and checks

- Extended the real CLI setup test to require `conductor-onboard/SKILL.md` in both Codex and Claude skill roots and its removal on uninstall. Before adding the skill it failed with `missing installed skill conductor-onboard`.
- Adding the embedded file alone still failed: `knownSkill` routed the unrecognized name beneath `conductor-work`. Adding the new name to the installer allowlist made the installation checks pass.
- Extended the existing interrupted bundle-upgrade test to verify the new skill survives journal-first recovery, reaches its intended root, and is removed by uninstall. Existing tests retain idempotence, user-edit conflict and configuration-preservation checks.
- Focused setup/upgrade tests passed. Full `go test ./... -count=1` passed across CLI, core, Git discovery, portal API, setup, store and integration packages. `go vet ./...` and a Windows executable build passed.
- The skill-creator frontmatter validator passed. Relative references resolve within the bundle; the existing embed pattern includes the new skill automatically.

Go was invoked from the repository-local `.tools/go/bin/go.exe`, with the existing `.tools/gocache` and `.tools/gomodcache`. Host access and tool-discovery failures are preserved in CON-P39. No shared database migration, production binary replacement or personal skill/configuration update was performed.

## Behavioral evaluation

An independent baseline agent used the existing work and plan skills for a fictional Prism project. It correctly preserved historical plans, separated implementation from outstanding hardware QA, reused an existing ticket from a later page, avoided unapproved future work and treated failed discovery as unknown registration. This baseline did not establish a broad behavioral failure. It identified missing concrete pagination guidance and no standard onboarding checkpoint shape.

A fresh evaluator then read the new skill and the same raw scenario: export implemented with hardware QA pending, live output approved and already tracked on page two, cloud sync unapproved, and an interrupted create with an exact saved request/payload but no response. It produced the expected source map and supported `ticket list --limit 100 --cursor ...` sequence without mutations or invented IDs.

The evaluator identified one ambiguity: replaying a lost create merely to discover provenance can duplicate a pre-existing matching ticket if the original request never committed. The skill now explicitly reuses matching work and excludes that old operation from pending retries while preserving its unknown provenance. It also resolves policy before creating the onboarding tracking ticket, so that ticket cannot accidentally precede a requested policy configuration.

The evaluator rechecked both changes and confirmed correct behavior for Prism and for a new project with requested independent-review policy. A separate code/contract reviewer found no actionable issues after inspecting CLI contracts, replay transactions, policy/coordinator restrictions, installer ownership and the scoped diff.

These evaluations are fresh-context reasoning trials, not an actual production backlog import. Installer tests execute real filesystem operations in temporary host roots; those results do not prove fresh live-host skill discovery.

## Lux read-only assessment

Both root and evaluator read Lux's document index, roadmap, implementation handoff and validation scope; the evaluator also read the transport checkpoint. Git discovery returned `DISCOVERY_UNAVAILABLE` due repository ownership. No global trust setting was changed. Registration, current checkout revision/dirty state, existing tickets, policy and active-team state remain unverified.

| Source in Lux | Observed documentation | Onboarding consequence |
| --- | --- | --- |
| `docs/README.md` and `docs/implementation/roadmap.md` | Current amendment moves installed export, Studio-free cold start and independent instances into tracer 0.1 | Follow current scope and linked DEC-13/export-scope record; preserve original plans |
| `docs/implementation/transport-status.md` and handoff | Documents transport implementation at `a5eee83` on `codex/resolume-transport`; status update did not merge application code | Inspect that branch and raw evidence before importing work; retain possible integration work rather than recreating transport |
| Transport checkpoint and roadmap 0.1 | Export/install, runtime provisioning, automatic startup, independent instances and full acceptance remain open | Map remaining implementation and validation separately after current-state inspection |
| Handoff and linked TR task plan | Describes preflight/core/AI/Studio responsibilities | Inspect their current implementation; a copied execution prompt is not new authorization |
| Roadmap 0.2–6 | Later milestones and extension contracts | Preserve summarized future scope; avoid speculative task expansion |
| `docs/validation.md` | Explicitly validates planning artifacts only | Do not present document checks as application or hardware acceptance |

Branch, test and host claims above are documented reports, not independently reverified implementation results. The assessment wrote no Lux files and created no Lux tickets. Actual onboarding should first resolve discovery, inspect the remaining authoritative sources and implementation, and use the user's selected scope/policy.
