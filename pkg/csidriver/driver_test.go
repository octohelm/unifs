package csidriver

import (
	"context"
	"path"
	"testing"

	"github.com/kubernetes-csi/csi-test/v5/pkg/sanity"

	"github.com/octohelm/x/logr"
	"github.com/octohelm/x/logr/slog"
)

func TestDriver(t *testing.T) {
	t.Skip()

	driver := newDriver(t)

	sanityCfg := sanity.NewTestConfig()
	sanityCfg.Address = driver.Endpoint
	sanityCfg.TargetPath = path.Join(t.TempDir(), "target")
	sanityCfg.StagingPath = path.Join(t.TempDir(), "staging")

	sanityCfg.SecretsFile = "testdata/secrets.yaml"

	sanity.Test(t, sanityCfg)
}

func newDriver(t *testing.T) *Driver {
	ctx := logr.WithLogger(context.Background(), slog.Logger(slog.Default()))

	socket := path.Join(t.TempDir(), "csi-driver.sock")

	driver := &Driver{}
	driver.Endpoint = "unix://" + socket
	driver.NodeID = "test-node"
	if err := driver.Init(ctx); err != nil {
		t.Fatal(err)
	}
	go func() {
		_ = driver.Serve(ctx)
	}()
	t.Cleanup(func() {
		_ = driver.Shutdown(ctx)
	})
	return driver
}
