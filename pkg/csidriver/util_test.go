package csidriver

import (
	"testing"

	"github.com/container-storage-interface/spec/lib/go/csi"
)

func newFakeDriverContext(t *testing.T) DriverContext {
	dctx := DriverContext{
		Name:          DefaultDriverName,
		VendorVersion: "v0.1.0",
		NodeID:        "test-node",
	}

	dctx.AddControllerServiceCapabilities(
		csi.ControllerServiceCapability_RPC_CREATE_DELETE_VOLUME,
	)

	dctx.AddVolumeCapabilityAccessModes(
		csi.VolumeCapability_AccessMode_SINGLE_NODE_WRITER,
	)

	return dctx
}
