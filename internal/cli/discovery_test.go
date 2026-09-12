package cli_test

import (
	"bytes"
	"context"
	"encoding/json"
	"github.com/ZackMFleischman/conductor/internal/cli"
	"os"
	"path/filepath"
	"testing"
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
