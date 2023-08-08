package local

import (
	"github.com/octohelm/unifs/pkg/filesystem/testutil"
	"testing"
)

func TestLocalFS(t *testing.T) {
	t.Run("Simple", func(t *testing.T) {
		testutil.TestSimpleFS(t, NewLocalFS(t.TempDir()))
	})

	t.Run("Full", func(t *testing.T) {
		testutil.TestFullFS(t, NewLocalFS(t.TempDir()))
	})
}
