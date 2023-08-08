package main

import (
	"context"
	"fmt"
	"github.com/go-courier/logr"
	"github.com/hanwen/go-fuse/v2/fs"
	fusefuse "github.com/hanwen/go-fuse/v2/fuse"
	"github.com/innoai-tech/infra/pkg/cli"
	"github.com/innoai-tech/infra/pkg/configuration"
	"github.com/innoai-tech/infra/pkg/otel"
	"github.com/octohelm/unifs/pkg/filesystem"
	"github.com/octohelm/unifs/pkg/filesystem/local"
	"github.com/octohelm/unifs/pkg/filesystem/s3"
	"github.com/octohelm/unifs/pkg/filesystem/webdav"
	"github.com/octohelm/unifs/pkg/fuse"
	"github.com/octohelm/unifs/pkg/strfmt"
	"github.com/pkg/errors"
	"os"
	"time"
)

func init() {
	cli.AddTo(App, &Mount{})
}

var _ configuration.Server = &Mounter{}

// Mount as fuse fs
type Mount struct {
	cli.C
	Otel otel.Otel

	Mounter
}

type Mounter struct {
	MountPoint string `arg:""`
	// Source Endpoint
	Endpoint strfmt.Endpoint `flag:"endpoint"`

	fsi   filesystem.FileSystem `flag:"-"`
	state *fusefuse.Server      `flag:"-"`
}

func (m *Mounter) Init(ctx context.Context) error {
	switch m.Endpoint.Scheme {
	case "s3":
		conf := &s3.Config{Endpoint: m.Endpoint}
		c, err := conf.Client(ctx)
		if err != nil {
			return err
		}
		m.fsi = s3.NewS3FS(c, conf.Bucket(), conf.Prefix())
	case "webdav":
		conf := &webdav.Config{Endpoint: m.Endpoint}
		c, err := conf.Client(ctx)
		if err != nil {
			return err
		}
		m.fsi = webdav.NewWebdavFS(c)
	case "file":
		m.fsi = local.NewLocalFS(m.Endpoint.Path)
	default:
		return errors.Errorf("unsupported endpoint %s", m.Endpoint)
	}

	return nil
}

func (m *Mounter) Serve(ctx context.Context) error {
	if err := os.MkdirAll(m.MountPoint, os.ModePerm); err != nil {
		return err
	}

	options := &fs.Options{}
	options.Name = fmt.Sprintf("%s.fs", m.Endpoint.Scheme)
	//options.Debug = true

	rawFS := fs.NewNodeFS(fuse.FS(m.fsi), options)

	state, err := fusefuse.NewServer(rawFS, m.MountPoint, &options.MountOptions)
	if err != nil {
		return err
	}
	m.state = state

	logr.FromContext(ctx).
		WithValues(
			"fsi", m.Endpoint.Scheme,
			"on", m.MountPoint,
		).
		Info("mounted")

	state.Serve()
	return nil
}

func (m *Mounter) Shutdown(ctx context.Context) error {
	if m.state == nil {
		return nil
	}

	errCh := make(chan error)

	go func() {
		for i := 0; i < 5; i++ {
			err := m.state.Unmount()
			if err == nil {
				errCh <- err
				return
			}
			logr.FromContext(ctx).Warn(errors.Wrap(err, "unmount failed"))
			time.Sleep(time.Second)
			logr.FromContext(ctx).Info("retrying...")
		}
		errCh <- m.state.Unmount()
	}()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case err := <-errCh:
		return err
	}
}
