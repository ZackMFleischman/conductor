package cli_test

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/ZackMFleischman/conductor/internal/cli"
	"github.com/ZackMFleischman/conductor/internal/store"
)

func TestServeHelpAndUnsafeListen(t *testing.T) {
	for _, c := range []struct {
		args []string
		code int
	}{
		{[]string{"serve", "--help"}, 0},
		{[]string{"serve", "--listen", "0.0.0.0:7331"}, 2},
		{[]string{"serve", "--listen", "[::]:7331"}, 2},
		{[]string{"serve", "--listen", "example.com:7331"}, 2},
		{[]string{"serve", "--project", "p"}, 2},
		{[]string{"serve"}, 5},
	} {
		t.Run(stringMustJSON(c.args), func(t *testing.T) {
			home := filepath.Join(t.TempDir(), "missing")
			var out bytes.Buffer
			code := cli.Run(context.Background(), cli.Env{Home: home, Out: &out, Err: io.Discard}, c.args)
			if code != c.code {
				t.Fatalf("exit%d: %s", code, out.String())
			}
			if _, e := os.Stat(home); !os.IsNotExist(e) {
				t.Fatal("serve must not initialize a registry")
			}
		})
	}
}
func stringMustJSON(v any) string { b, _ := json.Marshal(v); return string(b) }

type startupWriter struct{ ch chan []byte }

func (w startupWriter) Write(p []byte) (int, error) {
	w.ch <- append([]byte(nil), p...)
	return len(p), nil
}

func TestServeExistingReadOnlyStoreAndShutdown(t *testing.T) {
	home := t.TempDir()
	path := filepath.Join(home, "conductor.db")
	s, e := store.Open(path, true)
	if e != nil {
		t.Fatal(e)
	}
	if _, e = s.DB.Exec("INSERT INTO projects(id,common_dir,prefix) VALUES('p','fixture','APP')"); e != nil {
		t.Fatal(e)
	}
	s.DB.Close()
	before, e := os.ReadFile(path)
	if e != nil {
		t.Fatal(e)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	ready := startupWriter{make(chan []byte, 1)}
	done := make(chan int, 1)
	go func() {
		done <- cli.Run(ctx, cli.Env{Home: home, Out: io.Discard, Err: ready}, []string{"serve", "--listen", "127.0.0.1:0"})
	}()
	var address string
	select {
	case b := <-ready.ch:
		var v struct {
			Address string `json:"address"`
		}
		if e = json.Unmarshal(b, &v); e != nil {
			t.Fatal(e)
		}
		address = v.Address
	case c := <-done:
		t.Fatalf("server exited early %d", c)
	case <-time.After(5 * time.Second):
		t.Fatal("no ready event")
	}
	res, e := http.Get("http://" + address + "/api/v1/projects/p/board")
	if e != nil {
		t.Fatal(e)
	}
	body, _ := io.ReadAll(res.Body)
	res.Body.Close()
	if res.StatusCode != 200 || !bytes.Contains(body, []byte(`"name":"APP"`)) {
		t.Fatalf("response%d %s", res.StatusCode, body)
	}
	cancel()
	select {
	case c := <-done:
		if c != 0 {
			t.Fatalf("shutdown exit%d", c)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("shutdown stuck")
	}
	after, e := os.ReadFile(path)
	if e != nil {
		t.Fatal(e)
	}
	if !bytes.Equal(before, after) {
		t.Fatal("serve modified the registry")
	}
}
