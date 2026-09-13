package cli

import (
	"context"
	"encoding/json"
	"errors"
	"io/fs"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/ZackMFleischman/conductor/internal/core"
	"github.com/ZackMFleischman/conductor/internal/portalapi"
	"github.com/ZackMFleischman/conductor/internal/store"
)

func init() { Register("serve", Serve) }

func Serve(ctx context.Context, env Env, args []string) (any, error) {
	f, err := Parse(args, []string{"listen", "assets"}, []string{"help"})
	if err != nil {
		return nil, err
	}
	if err = f.NoPositionals(); err != nil {
		return nil, err
	}
	if f.Bools["help"] {
		return map[string]any{"usage": "conductor --home PATH serve [--listen 127.0.0.1:7331] [--assets DIR]", "description": "Read-only local portal API and live event stream; requires existing schema 3 registry"}, nil
	}
	if f.Values["project"] != "" {
		return nil, core.Fail("USAGE", "serve selects projects through the API; omit --project")
	}
	address := f.Values["listen"]
	if address == "" {
		address = "127.0.0.1:7331"
	}
	if err = portalapi.ValidateListen(address); err != nil {
		return nil, core.Fail("USAGE", err.Error())
	}
	var assets fs.FS
	if dir := f.Values["assets"]; dir != "" {
		root, e := os.OpenRoot(dir)
		if e != nil {
			return nil, core.Fail("USAGE", "assets must be an accessible directory")
		}
		defer root.Close()
		assets = root.FS() // Root confines symlinks/junctions to the explicitly supplied tree.
	}
	db, err := store.OpenReadOnly(filepath.Join(env.Home, "conductor.db"))
	if err != nil {
		return nil, err
	}
	defer db.DB.Close()
	var version int
	if err = db.DB.QueryRowContext(ctx, "PRAGMA user_version").Scan(&version); err != nil {
		return nil, err
	}
	if version != 3 {
		return nil, core.Fail("REGISTRY_UNAVAILABLE", "portal requires an existing schema 3 registry; upgrade with the CLI first")
	}
	listener, err := net.Listen("tcp", address)
	if err != nil {
		return nil, err
	}
	defer listener.Close()
	ctx, stop := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)
	defer stop()
	h := portalapi.NewHandler(&portalapi.DBReader{DB: db.DB}, portalapi.Options{Assets: assets})
	server := &http.Server{Handler: h, ReadHeaderTimeout: 5 * time.Second, IdleTimeout: 60 * time.Second, MaxHeaderBytes: 16 * 1024, BaseContext: func(net.Listener) context.Context { return ctx }}
	if err = json.NewEncoder(env.Err).Encode(map[string]string{"event": "server.started", "address": listener.Addr().String()}); err != nil {
		return nil, err
	}
	err = runPortalServer(ctx, server, listener)
	if errors.Is(err, http.ErrServerClosed) {
		return map[string]bool{"stopped": true}, nil
	}
	return nil, err
}
