package tar

import (
	"archive/tar"
	"context"
	"fmt"
	"io"
	"os"
	"path"

	"github.com/octohelm/unifs/pkg/filesystem"
)

type ImportOption func(t *tarImporter)

type Importer interface {
	ImportFrom(ctx context.Context, r io.Reader) error
}

func WithDest(dest string) ImportOption {
	return func(t *tarImporter) {
		t.dest = dest
	}
}

func To(fsys filesystem.FileSystem, opts ...ImportOption) Importer {
	ti := &tarImporter{
		fsys: fsys,
	}
	ti.Build(opts...)
	return ti
}

type tarImporter struct {
	fsys filesystem.FileSystem
	dest string
}

func (i *tarImporter) ImportFrom(ctx context.Context, r io.Reader) error {
	base := i.dest
	if base != "" {
		if err := filesystem.MkdirAll(ctx, i.fsys, base); err != nil {
			return fmt.Errorf("mkdir tar destination %q: %w", base, err)
		}
	}

	fullname := func(name string) string {
		if base != "" {
			return path.Join(base, name)
		}
		return name
	}

	tr := tar.NewReader(r)

	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("read tar header: %w", err)
		}

		filename := fullname(hdr.Name)

		if err := i.writeFile(ctx, filename, tr, hdr); err != nil {
			return fmt.Errorf("import tar entry %q: %w", hdr.Name, err)
		}
	}

	return nil
}

func (i *tarImporter) writeFile(ctx context.Context, filename string, r io.Reader, h *tar.Header) error {
	dir := path.Dir(filename)
	if dir != "" && dir != "." {
		if err := filesystem.MkdirAll(ctx, i.fsys, dir); err != nil {
			return fmt.Errorf("mkdir parent %q: %w", dir, err)
		}
	}
	f, err := i.fsys.OpenFile(ctx, filename, os.O_RDWR|os.O_TRUNC|os.O_CREATE, os.ModePerm)
	if err != nil {
		return fmt.Errorf("open %q for tar import: %w", filename, err)
	}
	defer f.Close()

	if _, err := io.CopyN(f, r, h.Size); err != nil {
		return fmt.Errorf("write tar entry %q: %w", filename, err)
	}
	return nil
}

func (i *tarImporter) Build(opts ...ImportOption) {
	for _, opt := range opts {
		opt(i)
	}
}
