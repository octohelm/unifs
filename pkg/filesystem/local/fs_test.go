package local

import (
	"testing"

	"github.com/octohelm/unifs/pkg/filesystem/testutil"
)

func TestLocalFS(t *testing.T) {
	t.Run("Simple", func(t *testing.T) {
		testutil.TestSimpleFS(t, NewFS(t.TempDir()))
	})

	t.Run("Full", func(t *testing.T) {
		testutil.TestFullFS(t, NewFS(t.TempDir()))
	})
}
