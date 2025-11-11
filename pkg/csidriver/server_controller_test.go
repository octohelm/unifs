package csidriver

import (
	"context"
	"reflect"
	"testing"

	"github.com/container-storage-interface/spec/lib/go/csi"

	"github.com/octohelm/x/logr/slog"
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
				// Setup
				cs := newFakeControllerServer(t)
				// Run
				resp, err := cs.CreateVolume(context.Background(), c.req)

				if !c.expectErr && err != nil {
					t.Errorf("c %q failed: %v", c.name, err)
				}
				if c.expectErr && err == nil {
					t.Errorf("c %q failed; got success", c.name)
				}

				if !reflect.DeepEqual(resp, c.resp) {
					t.Errorf("c %q failed: got resp %+v, expected %+v", c.name, resp, c.resp)
				}
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
