package api

import (
	"context"

	"github.com/octohelm/unifs/pkg/filesystem"
	"github.com/octohelm/unifs/pkg/filesystem/local"
	"github.com/octohelm/unifs/pkg/filesystem/s3"
	"github.com/octohelm/unifs/pkg/filesystem/webdav"
	"github.com/octohelm/unifs/pkg/strfmt"
	"github.com/pkg/errors"
)

type FileSystemBackend struct {
	Backend strfmt.Endpoint `flag:"backend"`

	fsi filesystem.FileSystem `flag:"-"`
}

func (m *FileSystemBackend) FileSystem() filesystem.FileSystem {
	return m.fsi
}

func (m *FileSystemBackend) Init(ctx context.Context) error {
	switch m.Backend.Scheme {
	case "s3":
		conf := &s3.Config{Endpoint: m.Backend}
		c, err := conf.Client(ctx)
		if err != nil {
			return err
		}
		m.fsi = s3.NewS3FS(c, conf.Bucket(), conf.Prefix())
	case "webdav":
		conf := &webdav.Config{Endpoint: m.Backend}
		c, err := conf.Client(ctx)
		if err != nil {
			return err
		}
		m.fsi = webdav.NewWebdavFS(c)
	case "file":
		m.fsi = local.NewLocalFS(m.Backend.Path)
	default:
		return errors.Errorf("unsupported %s", m.Backend)
	}
	return nil
}
