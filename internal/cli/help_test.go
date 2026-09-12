package cli_test

import (
	"github.com/ZackMFleischman/conductor/internal/testkit"
	"testing"
)

func TestHelpVersionAndProjectGlobal(t *testing.T) {
	c := testkit.New(t)
	testkit.MustData(t, c.Run("--help"))
	testkit.MustData(t, c.Run("--version"))
	p := testkit.MustData(t, c.Run("init", "--prefix", "APP", "--request", "i"))
	testkit.MustData(t, c.Run("--project", p["project_id"].(string), "agent", "list"))
}
