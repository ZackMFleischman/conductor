# Local agent host checks

Observed on Windows, 2026-09-12. This records actual checks, not planned acceptance.

| Check | Result |
| --- | --- |
| Go toolchain | Official Go 1.27.1 Windows amd64 archive downloaded; SHA-256 verified against the release manifest |
| Codex CLI | Version 0.154.0; signed in using ChatGPT |
| Claude Code | Version 2.1.217; reports signed out, including outside the parent execution sandbox |
| Foundation binary | Built from the T0 source subsequently committed as `68ad583` |
| Default shared-data access from parent sandbox | Correctly returned `REGISTRY_UNAVAILABLE` with the denied directory, without creating another database |
| Approved foundation probe | SQLite open, WAL creation, persistence, reopening, and cleanup all passed in the default shared data directory |
| Fresh Codex session, direct sandbox execution | Host refused to execute before the CLI ran; see limitation below |
| Fresh Codex session, normal command approval | The same probe passed all five checks in the default shared directory |
| Fresh-session automatic tracker discovery | Not yet tested; requires the integrated installer and work skill |
| Fresh Claude Code coordination | Not yet tested; requires sign-in |
| Linux runtime | Not tested; cross-compilation alone is not runtime acceptance |

The shared directory was `%LOCALAPPDATA%\Conductor`, outside all Git worktrees. The probe used only temporary files and cleaned them; it did not initialize a business database or register the project.

## Windows Codex limitation

The existing Codex configuration selects `windows.sandbox = "unelevated"`. A fresh session with the narrow shared directory added to its writable roots failed with:

```text
windows unelevated restricted-token sandbox cannot enforce split writable root sets directly; refusing to run unsandboxed
```

Using the host's normal per-command approval flow allowed the CLI probe to run. No sandbox configuration was changed and protections were not globally disabled. This establishes approved-command compatibility; it does not establish approval-free operation under that sandbox implementation. Installation must distinguish configured access from an actual successful probe.

A host process exiting successfully is not sufficient evidence: the initial test session exited successfully while reporting that its command never ran. Acceptance checks inspect the tool result and probe JSON.

## Sources for installation behavior

- [Codex user skills](https://learn.chatgpt.com/docs/build-skills)
- [Codex global instruction precedence](https://learn.chatgpt.com/docs/agent-configuration/agents-md)
- [Codex writable-root configuration](https://learn.chatgpt.com/docs/config-file/config-reference)
- [Claude Code personal skills](https://code.claude.com/docs/en/skills)
- [Claude Code configuration locations](https://code.claude.com/docs/en/settings)
