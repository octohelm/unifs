package api

import (
	"context"
	"github.com/octohelm/unifs/pkg/filesystem/ftp"

	"github.com/octohelm/unifs/pkg/filesystem"
	"github.com/octohelm/unifs/pkg/filesystem/local"
	"github.com/octohelm/unifs/pkg/filesystem/s3"
	"github.com/octohelm/unifs/pkg/filesystem/webdav"
	"github.com/octohelm/unifs/pkg/strfmt"
	"github.com/pkg/errors"
)

type FileSystemBackend struct {
	// 地址
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
		m.fsi = s3.NewFS(c, conf.Bucket(), conf.Prefix())
		return nil
	case "ftp", "ftps":
		m.fsi = ftp.NewFS(&ftp.Config{Endpoint: m.Backend})
		return nil
	case "webdav":
		conf := &webdav.Config{Endpoint: m.Backend}
		c, err := conf.Client(ctx)
		if err != nil {
			return err
		}
		m.fsi = webdav.NewFS(c)
		return nil
	case "file":
		m.fsi = local.NewFS(m.Backend.Path)
		return nil
	default:
		return errors.Errorf("unsupported %s", m.Backend)
	}
}

func (m *FileSystemBackend) InjectContext(ctx context.Context) context.Context {
	return filesystem.Context.Inject(ctx, m.fsi)
}
