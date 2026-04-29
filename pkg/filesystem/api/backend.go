package api

import (
	"context"
	"fmt"
	"net/url"
	"strings"

	"github.com/octohelm/unifs/pkg/filesystem"
	"github.com/octohelm/unifs/pkg/filesystem/ftp"
	"github.com/octohelm/unifs/pkg/filesystem/local"
	"github.com/octohelm/unifs/pkg/filesystem/s3"
	"github.com/octohelm/unifs/pkg/filesystem/webdav"
	"github.com/octohelm/unifs/pkg/strfmt"
)

// FileSystemBackend 根据端点配置并初始化文件系统后端。
type FileSystemBackend struct {
	// 地址
	Backend strfmt.Endpoint `flag:"backend,omitzero"`
	// 非空时覆盖用户名
	UsernameOverwrite string `flag:",omitzero"`
	// 非空时覆盖密码
	PasswordOverwrite string `flag:",omitzero,secret"`
	// 非空时覆盖路径
	PathOverwrite string `flag:",omitzero"`
	// 非空时覆盖 extra 查询参数
	ExtraOverwrite string `flag:",omitzero"`

	fsi filesystem.FileSystem `flag:"-"`
}

// Disabled 判断是否未配置后端端点。
func (m *FileSystemBackend) Disabled(ctx context.Context) bool {
	return m.Backend.IsZero()
}

// FileSystem 返回已初始化的文件系统。
func (m *FileSystemBackend) FileSystem() filesystem.FileSystem {
	return m.fsi
}

// Init 根据端点配置初始化文件系统后端。
func (m *FileSystemBackend) Init(ctx context.Context) error {
	if m.Disabled(ctx) {
		return nil
	}

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
			return fmt.Errorf("parse backend extra %q: %w", extra, err)
		}
		endpoint.Extra = q
	}

	switch endpoint.Scheme {
	case "s3":
		conf := &s3.Config{Endpoint: endpoint}
		fsys, err := conf.AsFileSystem(ctx)
		if err != nil {
			return fmt.Errorf("init s3 backend %s: %w", endpoint.SecurityString(), err)
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
			return fmt.Errorf("init webdav backend %s: %w", endpoint.SecurityString(), err)
		}
		m.fsi = webdav.NewFS(c)
		return nil
	case "file":
		if endpoint.Hostname == "." && strings.HasPrefix(endpoint.Path, "/") {
			m.fsi = local.NewFS(endpoint.Path[1:])
			return nil
		}
		m.fsi = local.NewFS(endpoint.Path)
		return nil
	default:
		return fmt.Errorf("unsupported %s", endpoint)
	}
}

// InjectContext 将已初始化的文件系统存入 ctx。
func (m *FileSystemBackend) InjectContext(ctx context.Context) context.Context {
	return filesystem.Context.Inject(ctx, m.fsi)
}
