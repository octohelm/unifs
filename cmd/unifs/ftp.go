package main

import (
	"github.com/innoai-tech/infra/pkg/cli"
	"github.com/innoai-tech/infra/pkg/otel"

	"github.com/octohelm/unifs/pkg/filesystem/api"
	"github.com/octohelm/unifs/pkg/ftp"
)

func init() {
	cli.AddTo(App, &Ftp{})
}

// Serve Webdav as fuse fs
type Ftp struct {
	cli.C
	Otel otel.Otel

	api.FileSystemBackend

	ftp.Server
}
