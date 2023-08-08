package main

import (
	"context"
	"os"

	"github.com/innoai-tech/infra/pkg/cli"

	"github.com/octohelm/unifs/internal/version"
)

var App = cli.NewApp("unifs", version.Version(), cli.WithImageNamespace("ghcr.io/octohelm"))

func main() {
	if err := cli.Execute(context.Background(), App, os.Args[1:]); err != nil {
		os.Exit(1)
	}
}
