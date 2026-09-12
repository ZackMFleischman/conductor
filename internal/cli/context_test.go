package cli_test

import (
	"github.com/ZackMFleischman/conductor/internal/testkit"
	"testing"
)

func TestContextIncludesBoundedUpdates(t *testing.T) {
	c := testkit.New(t)
	testkit.MustData(t, c.Run("init", "--prefix", "APP", "--request", "i"))
	d := testkit.MustData(t, c.Run("context"))
	if _, ok := d["recent_updates"].([]any); !ok {
		t.Fatal("missing recent updates", d)
	}
}
