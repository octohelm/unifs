package main

import (
	"github.com/innoai-tech/infra/pkg/cli"
	"github.com/innoai-tech/infra/pkg/otel"

	"github.com/octohelm/unifs/pkg/csidriver"
)

func init() {
	cli.AddTo(App, &CSIDriver{})
}

// CSIDriver 启动 CSI Driver 服务。
type CSIDriver struct {
	cli.C
	Otel otel.Otel

	csidriver.Driver
}
