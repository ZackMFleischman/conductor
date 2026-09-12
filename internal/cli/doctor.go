package cli

import (
	"context"
	"github.com/ZackMFleischman/conductor/internal/core"
	"github.com/ZackMFleischman/conductor/internal/store"
	"os"
	"path/filepath"
)

func Doctor(ctx context.Context, env Env, args []string) (any, error) {
	f, e := Parse(args, nil, []string{"probe-write"})
	if e != nil {
		return nil, e
	}
	if e = f.NoPositionals(); e != nil {
		return nil, e
	}
	if !f.Bools["probe-write"] {
		return nil, core.Fail("USAGE", "--probe-write is required")
	}
	checks := map[string]any{}
	if e = os.MkdirAll(env.Home, 0700); e != nil {
		return nil, &core.Fault{Code: "REGISTRY_UNAVAILABLE", Message: "data directory is inaccessible", Details: map[string]any{"path": env.Home, "cause": e.Error()}}
	}
	file, e := os.CreateTemp(env.Home, "conductor-probe-*.db")
	if e != nil {
		return nil, e
	}
	path := file.Name()
	file.Close()
	defer func() {
		for _, suffix := range []string{"", "-wal", "-shm"} {
			os.Remove(path + suffix)
		}
	}()
	s, e := store.Open(path, true)
	if e != nil {
		return nil, &core.Fault{Code: "STORAGE_ERROR", Message: "SQLite probe failed", Details: map[string]any{"path": path, "cause": e.Error()}}
	}
	checks["open"] = true
	_, e = s.DB.ExecContext(ctx, "CREATE TABLE probe_session(id TEXT PRIMARY KEY,last_seen_at TEXT NOT NULL); INSERT INTO probe_session VALUES('probe','persisted')")
	if e != nil {
		s.DB.Close()
		return nil, e
	}
	_, e = os.Stat(path + "-wal")
	checks["wal_created"] = e == nil
	s.DB.Close()
	s, e = store.Open(path, false)
	if e != nil {
		return nil, e
	}
	var value string
	e = s.DB.QueryRowContext(ctx, "SELECT last_seen_at FROM probe_session WHERE id='probe'").Scan(&value)
	s.DB.Close()
	if e != nil {
		return nil, e
	}
	checks["reopen"] = true
	checks["persistence"] = value == "persisted"
	for _, suffix := range []string{"", "-wal", "-shm"} {
		if e = os.Remove(path + suffix); e != nil && !os.IsNotExist(e) {
			return nil, e
		}
	}
	checks["cleanup"] = true
	return map[string]any{"path": filepath.Clean(env.Home), "probe_path": path, "checks": checks}, nil
}
