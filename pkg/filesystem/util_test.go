package filesystem_test

import (
	"context"
	"os"
	"testing"

	"github.com/octohelm/unifs/pkg/filesystem"
	"github.com/octohelm/unifs/pkg/filesystem/local"
	testingx "github.com/octohelm/x/testing"
)

func TestMkdirAll(t *testing.T) {
	tmp := t.TempDir()
	t.Cleanup(func() {
		_ = os.RemoveAll(tmp)
	})

	fs := local.NewFS(tmp)
	err := filesystem.MkdirAll(context.Background(), fs, "path/to/deep")
	testingx.Expect(t, err, testingx.Be[error](nil))
}
