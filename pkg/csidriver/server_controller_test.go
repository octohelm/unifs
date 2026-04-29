package csidriver

import (
	"context"
	"testing"

	"github.com/container-storage-interface/spec/lib/go/csi"

	"github.com/octohelm/x/logr/slog"
	. "github.com/octohelm/x/testing/v2"
)

func TestController(t *testing.T) {
	t.Run("CreateVolume", func(t *testing.T) {
		cases := []struct {
			name      string
			req       *csi.CreateVolumeRequest
			resp      *csi.CreateVolumeResponse
			expectErr bool
		}{
			{
				name: "valid defaults",
				req: &csi.CreateVolumeRequest{
					Name: "volume-name",
					VolumeCapabilities: []*csi.VolumeCapability{
						{
							AccessType: &csi.VolumeCapability_Mount{
								Mount: &csi.VolumeCapability_MountVolume{},
							},
							AccessMode: &csi.VolumeCapability_AccessMode{
								Mode: csi.VolumeCapability_AccessMode_MULTI_NODE_MULTI_WRITER,
							},
						},
					},
					Secrets: map[string]string{
						backend: "file:///tmp/local",
					},
					Parameters: map[string]string{},
				},
				resp: &csi.CreateVolumeResponse{
					Volume: &csi.Volume{
						VolumeId:      "file##tmp/local#volume-name",
						VolumeContext: map[string]string{},
					},
				},
			},
		}

		for _, c := range cases {
			t.Run(c.name, func(t *testing.T) {
				cs := newFakeControllerServer(t)

				if c.expectErr {
					Then(t, "创建卷返回预期错误",
						ExpectDo(func() error {
							_, err := cs.CreateVolume(context.Background(), c.req)
							return err
						}, ErrorNotIs(nil)),
					)
					return
				}

				Then(t, "创建卷返回预期响应",
					ExpectMustValue(func() (*csi.CreateVolumeResponse, error) {
						return cs.CreateVolume(context.Background(), c.req)
					}, Equal(c.resp)),
				)
			})
		}
	})
}

func newFakeControllerServer(t *testing.T) *controllerServer {
	dctx := newFakeDriverContext(t)

	return &controllerServer{
		DriverContext: dctx,
		l:             slog.Logger(slog.Default()),
	}
}
