package monitor

import (
	"os"
	"testing"

	"agents-admin/tests/testutil"
)

var c *testutil.E2EClient

func TestMain(m *testing.M) {
	var err error
	c, err = testutil.SetupE2EClient()
	if err != nil {
		os.Exit(0)
	}
	os.Exit(m.Run())
}
