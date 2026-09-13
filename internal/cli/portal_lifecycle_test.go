package cli

import (
	"context"
	"errors"
	"io"
	"net"
	"net/http"
	"sync"
	"testing"
	"time"
)

func TestPortalShutdownWaitsForActiveHandler(t *testing.T) {
	listener, e := net.Listen("tcp", "127.0.0.1:0")
	if e != nil {
		t.Fatal(e)
	}
	entered := make(chan struct{})
	release := make(chan struct{})
	var once sync.Once
	finish := func() { once.Do(func() { close(release) }) }
	defer finish()
	server := &http.Server{Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { close(entered); <-release; w.WriteHeader(204) })}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan error, 1)
	go func() { done <- runPortalServer(ctx, server, listener) }()
	go func() {
		res, e := http.Get("http://" + listener.Addr().String())
		if e == nil {
			io.Copy(io.Discard, res.Body)
			res.Body.Close()
		}
	}()
	select {
	case <-entered:
	case <-time.After(time.Second):
		t.Fatal("handler did not start")
	}
	cancel()
	select {
	case e := <-done:
		if errors.Is(e, http.ErrServerClosed) {
			t.Fatal("returned before active handler drained")
		}
		t.Fatal(e)
	case <-time.After(50 * time.Millisecond):
	}
	finish()
	select {
	case <-done:
	case <-time.After(3 * time.Second):
		t.Fatal("shutdown did not finish")
	}
}
