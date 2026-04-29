package logr

import (
	"context"
	iofs "io/fs"
	"os"
	"testing"

	xlogr "github.com/octohelm/x/logr"
	. "github.com/octohelm/x/testing/v2"

	"github.com/octohelm/unifs/pkg/filesystem"
)

func TestWrap(t *testing.T) {
	ctx := context.Background()
	wrapped := Wrap(filesystem.NewMemFS(), xlogr.Discard())

	Then(t, "记录并转发文件系统操作",
		ExpectMust(func() error {
			return wrapped.Mkdir(ctx, "dir", 0o755)
		}),
		ExpectMust(func() error {
			f, err := wrapped.OpenFile(ctx, "dir/file.txt", os.O_CREATE|os.O_RDWR, 0o644)
			if err != nil {
				return err
			}
			defer f.Close()
			_, err = f.Write([]byte("hello"))
			return err
		}),
		ExpectMust(func() error {
			return wrapped.Rename(ctx, "dir/file.txt", "dir/renamed.txt")
		}),
		ExpectMust(func() error {
			return wrapped.RemoveAll(ctx, "dir/renamed.txt")
		}),
	)

	Then(t, "错误也被原样返回",
		ExpectDo(func() error {
			_, err := wrapped.Stat(ctx, "missing")
			return err
		}, ErrorIs(iofs.ErrNotExist)),
	)
}
