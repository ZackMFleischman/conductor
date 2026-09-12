package cli_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"github.com/ZackMFleischman/conductor/internal/cli"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ZackMFleischman/conductor/internal/testkit"
)

func TestUnregisteredContextDoesNotCreateStore(t *testing.T) {
	home := filepath.Join(t.TempDir(), "home")
	var out bytes.Buffer
	code := cli.Run(context.Background(), cli.Env{CWD: t.TempDir(), Home: home, Out: &out, Err: &out}, []string{"context", "--if-registered", "--json"})
	var v struct {
		OK   bool `json:"ok"`
		Data struct {
			Registered bool `json:"registered"`
		} `json:"data"`
	}
	if err := json.Unmarshal(out.Bytes(), &v); err != nil {
		t.Fatal(err)
	}
	if code != 0 || !v.OK || v.Data.Registered {
		t.Fatalf("unexpected %d %s", code, out.String())
	}
	if _, err := os.Stat(filepath.Join(home, "conductor.db")); !os.IsNotExist(err) {
		t.Fatal(err)
	}
}

func TestContextDiscoveryWithoutGitHelper(t *testing.T) {
	if os.Getenv("CONDUCTOR_DISCOVERY_HELPER") != "1" {
		return
	}
	os.Exit(cli.Run(context.Background(), cli.Env{CWD: os.Getenv("CONDUCTOR_TEST_CWD"), Home: os.Getenv("CONDUCTOR_TEST_HOME")}, []string{"context", "--if-registered"}))
}

func TestRegisteredContextWithoutGitIsUnknown(t *testing.T) {
	c := testkit.New(t)
	testkit.MustData(t, c.Run("init", "--prefix", "APP", "--request", "init"))
	assertDiscoveryUnknown(t, c, []string{"PATH=" + t.TempDir()}, "git")
}

func TestRegisteredContextUnsafeCheckoutIsUnknown(t *testing.T) {
	c := testkit.New(t)
	testkit.MustData(t, c.Run("init", "--prefix", "APP", "--request", "init"))
	assertDiscoveryUnknown(t, c, []string{"GIT_TEST_ASSUME_DIFFERENT_OWNER=1", "GIT_CONFIG_COUNT=1", "GIT_CONFIG_KEY_0=safe.directory", "GIT_CONFIG_VALUE_0="}, "dubious ownership")
}

func TestContextInaccessibleLocationIsUnknown(t *testing.T) {
	c := testkit.New(t)
	testkit.MustData(t, c.Run("init", "--prefix", "APP", "--request", "init"))
	c.CWD = filepath.Join(t.TempDir(), "missing")
	assertDiscoveryUnknown(t, c, nil, "git")
}

func assertDiscoveryUnknown(t *testing.T, c testkit.CLI, overrides []string, diagnostic string) {
	t.Helper()
	exe, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command(exe, "-test.run=^TestContextDiscoveryWithoutGitHelper$")
	for _, entry := range os.Environ() {
		replaced := false
		for _, override := range overrides {
			if strings.EqualFold(strings.SplitN(entry, "=", 2)[0], strings.SplitN(override, "=", 2)[0]) {
				replaced = true
			}
		}
		if !replaced {
			cmd.Env = append(cmd.Env, entry)
		}
	}
	cmd.Env = append(cmd.Env, overrides...)
	cmd.Env = append(cmd.Env, "CONDUCTOR_DISCOVERY_HELPER=1", "CONDUCTOR_TEST_CWD="+c.CWD, "CONDUCTOR_TEST_HOME="+c.Home)
	out, err := cmd.Output()
	v := testkit.Decode(t, out)
	var exit *exec.ExitError
	if !errors.As(err, &exit) || exit.ExitCode() != 5 || v["ok"] != false {
		t.Fatalf("discovery must fail with unknown registration: %v %v", err, v)
	}
	fault := v["error"].(map[string]any)
	if fault["code"] != "DISCOVERY_UNAVAILABLE" || !strings.Contains(fault["message"].(string), diagnostic) {
		t.Fatalf("missing discovery diagnostic: %v", v)
	}
}

func TestHealthyRegistryNonGitAndUnregisteredCheckout(t *testing.T) {
	c := testkit.New(t)
	testkit.MustData(t, c.Run("init", "--prefix", "APP", "--request", "init"))
	clone := filepath.Join(t.TempDir(), "clone")
	cmd := exec.Command("git", "clone", "--local", "-q", filepath.ToSlash(c.CWD), clone)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("clone: %s %v", out, err)
	}
	for _, cwd := range []string{t.TempDir(), clone} {
		c.CWD = cwd
		d := testkit.MustData(t, c.Run("context", "--if-registered"))
		if d["registered"] != false {
			t.Fatalf("expected unregistered: %v", d)
		}
	}
}
