package main

import (
	"context"
	"github.com/go-courier/logr"
	"github.com/innoai-tech/infra/pkg/cli"
	"github.com/innoai-tech/infra/pkg/configuration"
	"github.com/innoai-tech/infra/pkg/otel"
	"github.com/pkg/errors"
	netwebdav "golang.org/x/net/webdav"
	"net/http"
	"runtime"

	"github.com/octohelm/unifs/pkg/filesystem"
	"github.com/octohelm/unifs/pkg/filesystem/local"
	"github.com/octohelm/unifs/pkg/filesystem/s3"
	"github.com/octohelm/unifs/pkg/filesystem/webdav"
	"github.com/octohelm/unifs/pkg/strfmt"
)

func init() {
	cli.AddTo(App, &WebDAV{})
}

var _ configuration.Server = &WebDAV{}

// Serve Webdav as fuse fs
type WebDAV struct {
	cli.C
	Otel otel.Otel

	WebDAVServer
}

type WebDAVServer struct {
	// Source Endpoint
	Endpoint strfmt.Endpoint `flag:"endpoint"`

	Addr string `flag:"addr,omitempty"`

	fsi filesystem.FileSystem `flag:"-"`
	svc *http.Server          `flag:"-"`
}

func (s *WebDAVServer) SetDefaults() {
	if s.Addr == "" {
		s.Addr = ":8081"
	}
}

func (s *WebDAVServer) Init(ctx context.Context) error {
	switch s.Endpoint.Scheme {
	case "s3":
		conf := &s3.Config{Endpoint: s.Endpoint}
		c, err := conf.Client(ctx)
		if err != nil {
			return err
		}
		s.fsi = s3.NewS3FS(c, conf.Bucket(), conf.Prefix())
	case "webdav":
		conf := &webdav.Config{Endpoint: s.Endpoint}
		c, err := conf.Client(ctx)
		if err != nil {
			return err
		}
		s.fsi = webdav.NewWebdavFS(c)
	case "file":
		s.fsi = local.NewLocalFS(s.Endpoint.Path)
	default:
		return errors.Errorf("unsupported endpoint %s", s.Endpoint)
	}
	return nil
}

func (s *WebDAVServer) Serve(ctx context.Context) error {
	h := &netwebdav.Handler{
		FileSystem: s.fsi,
		LockSystem: netwebdav.NewMemLS(),
	}

	s.svc = &http.Server{
		Addr:    s.Addr,
		Handler: h,
	}

	logr.FromContext(ctx).Info("serve on %s (%s/%s)", s.svc.Addr, runtime.GOOS, runtime.GOARCH)

	return s.svc.ListenAndServe()
}

func (s *WebDAVServer) Shutdown(ctx context.Context) error {
	return s.svc.Shutdown(ctx)
}
