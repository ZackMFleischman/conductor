# Host access evidence

Foundation runtime: Windows amd64, Go 1.27.1 (`go version go1.27.1 windows/amd64`), modernc.org/sqlite v1.58.0. Go module dependencies are pinned in go.mod/go.sum; the setup parser is github.com/pelletier/go-toml/v2 v2.4.3.

The foundation tests exercise a real SQLite database, WAL creation, committed session-shaped data, close/reopen, read-back, and probe-owned cleanup. No business registry is created by doctor. `doctor --probe-write --json` reports the resolved absolute data directory and individual checks.

The parent integration task owns fresh-host evidence. Its initial observations: the managed Codex sandbox denies the default `%LOCALAPPDATA%\Conductor` directory, producing REGISTRY_UNAVAILABLE rather than silently selecting another home. An approved probe of the same binary and path succeeds for open, WAL, persistence, reopen, and cleanup. Fresh Codex execution with exact additional-directory access is under evaluation. Claude Code is installed but signed out, so its fresh-host acceptance remains open.

Git linked-worktree and independent-local-clone fixture validation passes under approved execution. Inside the managed sandbox, Git clone fails creating its subprocess signal pipe (Win32 error 5); this is a host restriction, not a skipped passing test.

Linux amd64 cross-compilation is checked separately. Linux runtime acceptance is not claimed.
