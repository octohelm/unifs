package main

import (
	"context"
	"net/http"
	"runtime"
	"time"

	"github.com/go-courier/logr"
	"github.com/innoai-tech/infra/pkg/cli"
	"github.com/innoai-tech/infra/pkg/configuration"
	"github.com/innoai-tech/infra/pkg/otel"
	"github.com/octohelm/unifs/pkg/filesystem/api"
	netwebdav "golang.org/x/net/webdav"
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
	Addr string `flag:"addr,omitzero"`

	api.FileSystemBackend

	svc *http.Server `flag:"-"`
}

func (s *WebDAVServer) SetDefaults() {
	if s.Addr == "" {
		s.Addr = ":8081"
	}
}

func (s *WebDAVServer) Serve(ctx context.Context) error {
	h := &netwebdav.Handler{
		FileSystem: s.FileSystem(),
		LockSystem: netwebdav.NewMemLS(),
	}

	s.svc = &http.Server{
		Addr:              s.Addr,
		ReadHeaderTimeout: 10 * time.Second,
		Handler:           h,
	}

	logr.FromContext(ctx).Info("serve on %s (%s/%s)", s.svc.Addr, runtime.GOOS, runtime.GOARCH)

	return s.svc.ListenAndServe()
}

func (s *WebDAVServer) Shutdown(ctx context.Context) error {
	return s.svc.Shutdown(ctx)
}
