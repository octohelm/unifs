package api

import (
	"context"
	"fmt"
	"net/url"

	"github.com/octohelm/unifs/pkg/filesystem"
	"github.com/octohelm/unifs/pkg/filesystem/ftp"
	"github.com/octohelm/unifs/pkg/filesystem/local"
	"github.com/octohelm/unifs/pkg/filesystem/s3"
	"github.com/octohelm/unifs/pkg/filesystem/webdav"
	"github.com/octohelm/unifs/pkg/strfmt"
)

type FileSystemBackend struct {
	// 地址
	Backend strfmt.Endpoint `flag:"backend"`
	// Overwrite username when not empty
	UsernameOverwrite string `flag:",omitempty"`
	// Overwrite password when not empty
	PasswordOverwrite string `flag:",omitempty,secret"`
	// Overwrite path when not empty
	PathOverwrite string `flag:",omitempty"`
	// Overwrite extra when not empty
	ExtraOverwrite string `flag:",omitempty"`

	fsi filesystem.FileSystem `flag:"-"`
}

func (m *FileSystemBackend) FileSystem() filesystem.FileSystem {
	return m.fsi
}

func (m *FileSystemBackend) Init(ctx context.Context) error {
	endpoint := m.Backend

	if path := m.PathOverwrite; path != "" {
		endpoint.Path = path
	}

	if username := m.UsernameOverwrite; username != "" {
		endpoint.Username = username
	}

	if password := m.PasswordOverwrite; password != "" {
		endpoint.Password = password
	}

	if extra := m.ExtraOverwrite; extra != "" {
		q, err := url.ParseQuery(extra)
		if err != nil {
			return err
		}
		endpoint.Extra = q
	}

	switch endpoint.Scheme {
	case "s3":
		conf := &s3.Config{Endpoint: endpoint}
		fsys, err := conf.AsFileSystem(ctx)
		if err != nil {
			return err
		}
		m.fsi = fsys
		return nil
	case "ftp", "ftps":
		m.fsi = ftp.NewFS(&ftp.Config{Endpoint: endpoint})
		return nil
	case "webdav":
		conf := &webdav.Config{Endpoint: endpoint}
		c, err := conf.Client(ctx)
		if err != nil {
			return err
		}
		m.fsi = webdav.NewFS(c)
		return nil
	case "file":
		m.fsi = local.NewFS(endpoint.Path)
		return nil
	default:
		return fmt.Errorf("unsupported %s", endpoint)
	}
}

func (m *FileSystemBackend) InjectContext(ctx context.Context) context.Context {
	return filesystem.Context.Inject(ctx, m.fsi)
}
