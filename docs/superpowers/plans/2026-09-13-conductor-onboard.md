# Existing-project onboarding implementation plan

**Goal:** Ship `conductor-onboard` through the existing skill bundle, using the current CLI.

**Approved design:** The user approved a skill that preserves existing documentation, reconciles current commitments against implementation evidence, registers only within authorization, establishes policy before importing remaining work, records source-to-ticket coverage, resumes without duplicate tickets, and hands off to existing planning and orchestration skills.

**Architecture:** One skill entrypoint uses sibling command references. The existing Go embed pattern and installer supply it to Codex and Claude. No database schema, importer command, or automatic dispatch is added.

## Work and validation

- [x] Read packaging, installed command contracts, existing planning guidance, and Lux's current document index, roadmap, implementation handoff and validation scope.
- [x] Run an independent baseline scenario using existing skills. Preserve findings in the verification report; do not characterize correct baseline behavior as failure.
- [x] Extend `internal/cli/setup_test.go` to require onboarding installation and removal for both hosts; observe failure before adding the skill.
- [x] Add `skills/conductor-onboard/SKILL.md` with a durable source mapping and request/payload checkpoint procedure. Include pagination syntax missing from existing planning references.
- [x] Add routing to `skills/conductor-plan/SKILL.md` and usage in README/quickstart. Keep existing project documents authoritative.
- [x] Run skill validation, setup tests and Go checks. Verify upgrades using the existing installer ownership tests and embedded bundle.
- [x] Run fresh-context behavioral evaluation plus independent review. Record the Lux assessment as read-only, including unverified registration and branch/evidence limitations.

## Delivery

Deliver the scoped work as a committed branch, submit CON-29 with evidence, and stop its session. Production installation and actual Lux backlog import are separate from this skill-authoring change. Verification results are in [onboarding skill acceptance](../../testing/onboarding-skill-acceptance.md); tracker submission records the exact delivered commit.
