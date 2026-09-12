package testkit

import (
	"bytes"
	"context"
	"encoding/json"
	"github.com/ZackMFleischman/conductor/internal/cli"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

type CLI struct {
	T         testing.TB
	CWD, Home string
	Binary    string
}

func New(t testing.TB) CLI {
	t.Helper()
	root := t.TempDir()
	cwd := filepath.Join(root, "repo")
	if e := os.Mkdir(cwd, 0700); e != nil {
		t.Fatal(e)
	}
	c := exec.Command("git", "init", "-q", cwd)
	if out, e := c.CombinedOutput(); e != nil {
		t.Fatalf("git init: %s %v", out, e)
	}
	return CLI{T: t, CWD: cwd, Home: filepath.Join(root, "home")}
}
func Decode(t testing.TB, b []byte) map[string]any {
	t.Helper()
	d := json.NewDecoder(bytes.NewReader(b))
	var v map[string]any
	if e := d.Decode(&v); e != nil {
		t.Fatalf("invalid envelope %q: %v", b, e)
	}
	var extra any
	if e := d.Decode(&extra); e != io.EOF {
		t.Fatalf("extra output %q", b)
	}
	if _, ok := v["ok"].(bool); !ok {
		t.Fatalf("missing ok: %v", v)
	}
	return v
}
func (c CLI) Run(args ...string) map[string]any {
	c.T.Helper()
	var out, errout bytes.Buffer
	cli.Run(context.Background(), cli.Env{CWD: c.CWD, Home: c.Home, Out: &out, Err: &errout}, args)
	return Decode(c.T, out.Bytes())
}
func MustData(t testing.TB, v map[string]any) map[string]any {
	t.Helper()
	if v["ok"] != true {
		t.Fatalf("command failed: %v", v)
	}
	d, ok := v["data"].(map[string]any)
	if !ok {
		t.Fatalf("not object data: %v", v)
	}
	return d
}
func String(t testing.TB, v map[string]any, k string) string {
	t.Helper()
	s, ok := v[k].(string)
	if !ok {
		t.Fatalf("%s not string: %v", k, v)
	}
	return s
}
func Number(t testing.TB, v map[string]any, k string) float64 {
	t.Helper()
	n, ok := v[k].(float64)
	if !ok {
		t.Fatalf("%s not number: %v", k, v)
	}
	return n
}
