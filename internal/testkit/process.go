package testkit

import (
	"os"
	"os/exec"
	"strings"
	"testing"
)

func Build(t testing.TB, root, binary string) {
	t.Helper()
	cmd := exec.Command("go", "build", "-o", binary, "./cmd/conductor")
	cmd.Dir = root
	if b, e := cmd.CombinedOutput(); e != nil {
		t.Fatalf("build %s: %v", b, e)
	}
}
func (c CLI) Process(args ...string) (map[string]any, int) {
	c.T.Helper()
	if c.Binary == "" {
		c.T.Fatal("set CLI.Binary to prebuilt binary")
	}
	cmd := exec.Command(c.Binary, args...)
	cmd.Dir = c.CWD
	for _, v := range os.Environ() {
		if !strings.HasPrefix(strings.ToUpper(v), "CONDUCTOR_HOME=") {
			cmd.Env = append(cmd.Env, v)
		}
	}
	cmd.Env = append(cmd.Env, "CONDUCTOR_HOME="+c.Home)
	b, e := cmd.Output()
	code := 0
	if e != nil {
		if ex, ok := e.(*exec.ExitError); ok {
			code = ex.ExitCode()
		} else {
			c.T.Fatal(e)
		}
	}
	return Decode(c.T, b), code
}
