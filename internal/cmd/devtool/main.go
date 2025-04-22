package main

import (
	"context"
	"os"

	"github.com/innoai-tech/infra/devpkg/gengo"
	"github.com/innoai-tech/infra/pkg/cli"
	"github.com/innoai-tech/infra/pkg/otel"

	_ "github.com/octohelm/gengo/devpkg/deepcopygen"
	_ "github.com/octohelm/gengo/devpkg/runtimedocgen"
)

var App = cli.NewApp("gengo", "dev")

func init() {
	cli.AddTo(App, &struct {
		cli.C `name:"gen"`
		otel.Otel
		gengo.Gengo
	}{})
}

func main() {
	if err := cli.Execute(context.Background(), App, os.Args[1:]); err != nil {
		panic(err)
	}
}
