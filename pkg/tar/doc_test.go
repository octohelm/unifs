package tar

import (
	"context"
	"os"
	"path"
	"testing"

	. "github.com/octohelm/x/testing/v2"

	"github.com/octohelm/unifs/pkg/filesystem/local"
)

func TestWrite(t *testing.T) {
	tmpDir := t.TempDir()

	t.Cleanup(func() {
		_ = os.RemoveAll(tmpDir)
	})

	fs := local.NewFS(".")

	t.Run("could write as tar", func(t *testing.T) {
		tarFile := path.Join(tmpDir, "x.tar")

		f := MustValue(t, func() (*os.File, error) {
			return os.OpenFile(tarFile, os.O_RDWR|os.O_TRUNC|os.O_CREATE, os.ModePerm)
		})

		Then(t, "导出 tar 成功",
			ExpectMust(func() error {
				return From(fs, WithBase("testdata/src")).ExportAsTar(context.Background(), f)
			}),
		)

		_ = f.Close()

		t.Run("then should import", func(t *testing.T) {
			f := MustValue(t, func() (*os.File, error) {
				return os.OpenFile(tarFile, os.O_RDONLY, os.ModePerm)
			})
			defer f.Close()

			i := To(fs, WithDest("testdata/dest"))

			Then(t, "从 tar 导入成功",
				ExpectMust(func() error {
					return i.ImportFrom(context.Background(), f)
				}),
			)
		})
	})
}

func TestWithImport(t *testing.T) {
	_ = From(local.NewFS("."), WithImport("testdata/src"))
}
