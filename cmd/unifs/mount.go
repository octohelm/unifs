package main

import (
	"context"
	"fmt"
	"github.com/octohelm/unifs/pkg/csidriver/mounter"
	"os"

	"github.com/go-courier/logr"
	"github.com/hanwen/go-fuse/v2/fs"
	fusefuse "github.com/hanwen/go-fuse/v2/fuse"
	"github.com/innoai-tech/infra/pkg/cli"
	"github.com/innoai-tech/infra/pkg/configuration"
	"github.com/innoai-tech/infra/pkg/otel"
	"github.com/octohelm/unifs/pkg/filesystem/api"
	"github.com/octohelm/unifs/pkg/fuse"
	"github.com/octohelm/unifs/pkg/strfmt"
	daemon "github.com/sevlyar/go-daemon"
)

func init() {
	cli.AddTo(App, &Mount{})
}

// Mount as fuse fs
type Mount struct {
	cli.C
	Otel otel.Otel

	Mounter
}

var _ configuration.Runner = &Mounter{}

type Mounter struct {
	MountPoint string          `arg:""`
	Backend    strfmt.Endpoint `flag:"backend"`
	Foreground bool            `flag:"foreground,omitempty"`
	Delegate   bool            `flag:"delegate,omitempty"`
}

func (m *Mounter) Run(ctx context.Context) error {
	if m.Delegate {
		m2, err := mounter.NewMounter(ctx, m.Backend.String())
		if err != nil {
			return err
		}
		return m2.Mount(m.MountPoint)
	}

	if !m.Foreground {
		dctx := &daemon.Context{}
		p, err := dctx.Reborn()
		if err != nil {
			return err
		}

		if p != nil {
			return nil
		}

		defer dctx.Release()
	}

	if err := os.MkdirAll(m.MountPoint, os.ModePerm); err != nil {
		return err
	}

	b := &api.FileSystemBackend{}
	b.Backend = m.Backend

	if err := b.Init(ctx); err != nil {
		return err
	}

	options := &fs.Options{}
	options.Name = fmt.Sprintf("%s.fs", b.Backend.Scheme)
	//options.Debug = true

	rawFS := fs.NewNodeFS(fuse.FS(b.FileSystem()), options)

	state, err := fusefuse.NewServer(rawFS, m.MountPoint, &options.MountOptions)
	if err != nil {
		return err
	}

	logr.FromContext(ctx).
		WithValues(
			"fsi", m.Backend.Scheme,
			"on", m.MountPoint,
		).
		Info("mounted")

	if !m.Foreground {
		go state.Serve()
		return daemon.ServeSignals()
	}

	state.Serve()
	return nil
}
