package local

import (
	"github.com/octohelm/unifs/pkg/filesystem/testutil"
	"testing"
)

func TestLocalFS(t *testing.T) {
	fs := NewLocalFS(t.TempDir())
	testutil.TestFS(t, fs)
}
