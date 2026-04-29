package filesystem_test

import (
	"context"
	"os"
	"testing"

	. "github.com/octohelm/x/testing/v2"

	"github.com/octohelm/unifs/pkg/filesystem"
	"github.com/octohelm/unifs/pkg/filesystem/local"
)

func TestMkdirAll(t *testing.T) {
	tmp := t.TempDir()
	t.Cleanup(func() {
		_ = os.RemoveAll(tmp)
	})

	fs := local.NewFS(tmp)

	Then(t, "创建所有缺失父目录",
		ExpectMust(func() error {
			return filesystem.MkdirAll(context.Background(), fs, "path/to/deep")
		}),
	)
}
