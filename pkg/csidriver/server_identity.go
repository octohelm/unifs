package csidriver

import (
	"context"

	"github.com/container-storage-interface/spec/lib/go/csi"
	"github.com/go-courier/logr"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

type identityServer struct {
	csi.UnsafeIdentityServer

	DriverContext

	l logr.Logger
}

func (i *identityServer) GetPluginInfo(ctx context.Context, request *csi.GetPluginInfoRequest) (*csi.GetPluginInfoResponse, error) {
	return &csi.GetPluginInfoResponse{
		Name:          i.Name,
		VendorVersion: i.VendorVersion,
	}, nil
}

func (i *identityServer) GetPluginCapabilities(ctx context.Context, request *csi.GetPluginCapabilitiesRequest) (*csi.GetPluginCapabilitiesResponse, error) {
	return &csi.GetPluginCapabilitiesResponse{
		Capabilities: []*csi.PluginCapability{
			{
				Type: &csi.PluginCapability_Service_{
					Service: &csi.PluginCapability_Service{
						Type: csi.PluginCapability_Service_CONTROLLER_SERVICE,
					},
				},
			},
		},
	}, nil
}

func (i *identityServer) Probe(ctx context.Context, request *csi.ProbeRequest) (*csi.ProbeResponse, error) {
	return &csi.ProbeResponse{Ready: &wrapperspb.BoolValue{Value: true}}, nil
}
