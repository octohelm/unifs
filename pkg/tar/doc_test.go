package tar

import (
	"context"
	"os"
	"path"
	"testing"

	"github.com/octohelm/unifs/pkg/filesystem/local"
	testingx "github.com/octohelm/x/testing"
)

func TestWrite(t *testing.T) {
	tmpDir := t.TempDir()

	t.Cleanup(func() {
		_ = os.RemoveAll(tmpDir)
	})

	fs := local.NewFS(".")

	t.Run("could write as tar", func(t *testing.T) {
		tarFile := path.Join(tmpDir, "x.tar")

		f, err := os.OpenFile(tarFile, os.O_RDWR|os.O_TRUNC|os.O_CREATE, os.ModePerm)
		testingx.Expect(t, err, testingx.BeNil[error]())

		err = From(fs, WithBase("testdata/src")).ExportAsTar(context.Background(), f)
		testingx.Expect(t, err, testingx.BeNil[error]())

		_ = f.Close()

		t.Run("then should import", func(t *testing.T) {
			f, err := os.OpenFile(tarFile, os.O_RDONLY, os.ModePerm)
			testingx.Expect(t, err, testingx.BeNil[error]())
			defer f.Close()

			i := To(fs, WithDest("testdata/dest"))
			testingx.Expect(t, err, testingx.BeNil[error]())

			err = i.ImportFrom(context.Background(), f)
			testingx.Expect(t, err, testingx.BeNil[error]())
		})
	})
}
