package tar

import (
	"archive/tar"
	"context"
	"io"
	"io/fs"
	"os"

	"cuelang.org/go/pkg/path"

	"github.com/octohelm/unifs/pkg/filesystem"
)

func WithBase(base string) ExportOption {
	return func(t *tarExporter) {
		t.base = base
	}
}

type ExportOption func(t *tarExporter)

type Exporter interface {
	ExportAsTar(ctx context.Context, w io.Writer) error
}

func From(fsys filesystem.FileSystem, opts ...ExportOption) Exporter {
	te := &tarExporter{
		fsys: fsys,
	}
	te.Build(opts...)
	return te
}

type tarExporter struct {
	fsys filesystem.FileSystem
	base string
}

func (t *tarExporter) Build(optFns ...ExportOption) {
	for _, x := range optFns {
		x(t)
	}
}

func (t *tarExporter) ExportAsTar(ctx context.Context, w io.Writer) error {
	tw := tar.NewWriter(w)
	defer tw.Close()

	base := t.base
	if base == "" {
		base = "."
	}

	return filesystem.WalkDir(ctx, t.fsys, base, func(pathname string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}

		if d.IsDir() {
			return nil
		}

		rel, err := path.Rel(base, pathname, path.Unix)
		if err != nil {
			return err
		}

		info, err := d.Info()
		if err != nil {
			return err
		}

		f, err := t.fsys.OpenFile(ctx, pathname, os.O_RDONLY, os.ModePerm)
		if err != nil {
			return err
		}
		defer f.Close()

		h := tar.Header{
			Name: rel,
			Size: info.Size(),
		}

		return t.writeToTar(tw, h, f)
	})
}

func (t *tarExporter) writeToTar(tw *tar.Writer, header tar.Header, r io.Reader) error {
	header.Mode = 0o644
	if err := tw.WriteHeader(&header); err != nil {
		return err
	}
	if _, err := io.CopyN(tw, r, header.Size); err != nil {
		return err
	}
	return nil
}
