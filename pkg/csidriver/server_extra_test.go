package csidriver

import (
	"context"
	"regexp"
	"testing"

	"github.com/container-storage-interface/spec/lib/go/csi"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/octohelm/x/logr"
	. "github.com/octohelm/x/testing/v2"
)

func TestIdentityAndNodeServer(t *testing.T) {
	ctx := context.Background()
	dctx := newFakeDriverContext(t)
	identity := &identityServer{DriverContext: dctx, l: logr.Discard()}
	node := &nodeServer{DriverContext: dctx, l: logr.Discard()}

	info := MustValue(t, func() (*csi.GetPluginInfoResponse, error) {
		return identity.GetPluginInfo(ctx, &csi.GetPluginInfoRequest{})
	})
	caps := MustValue(t, func() (*csi.GetPluginCapabilitiesResponse, error) {
		return identity.GetPluginCapabilities(ctx, &csi.GetPluginCapabilitiesRequest{})
	})
	probe := MustValue(t, func() (*csi.ProbeResponse, error) {
		return identity.Probe(ctx, &csi.ProbeRequest{})
	})
	nodeCaps := MustValue(t, func() (*csi.NodeGetCapabilitiesResponse, error) {
		return node.NodeGetCapabilities(ctx, &csi.NodeGetCapabilitiesRequest{})
	})
	nodeInfo := MustValue(t, func() (*csi.NodeGetInfoResponse, error) {
		return node.NodeGetInfo(ctx, &csi.NodeGetInfoRequest{})
	})
	stats := MustValue(t, func() (*csi.NodeGetVolumeStatsResponse, error) {
		return node.NodeGetVolumeStats(ctx, &csi.NodeGetVolumeStatsRequest{})
	})

	Then(t, "identity 和 node 基础响应",
		Expect(info.Name, Equal(DefaultDriverName)),
		Expect(caps.Capabilities[0].GetService().Type, Equal(csi.PluginCapability_Service_CONTROLLER_SERVICE)),
		Expect(probe.Ready.Value, Equal(true)),
		Expect(nodeCaps.Capabilities[0].GetRpc().Type, Equal(csi.NodeServiceCapability_RPC_UNKNOWN)),
		Expect(nodeInfo.NodeId, Equal("test-node")),
		Expect(stats, Equal(&csi.NodeGetVolumeStatsResponse{})),
	)
}

func TestControllerAndNodeErrors(t *testing.T) {
	ctx := context.Background()
	c := &controllerServer{DriverContext: newFakeDriverContext(t), l: logr.Discard()}
	n := &nodeServer{DriverContext: newFakeDriverContext(t), l: logr.Discard()}

	controllerCaps := MustValue(t, func() (*csi.ControllerGetCapabilitiesResponse, error) {
		return c.ControllerGetCapabilities(ctx, &csi.ControllerGetCapabilitiesRequest{})
	})
	deleted := MustValue(t, func() (*csi.DeleteVolumeResponse, error) {
		return c.DeleteVolume(ctx, &csi.DeleteVolumeRequest{})
	})

	Then(t, "controller 基础响应",
		Expect(len(controllerCaps.Capabilities), Equal(1)),
		Expect(deleted, Equal(&csi.DeleteVolumeResponse{})),
	)

	Then(t, "controller 校验错误",
		ExpectDo(func() error {
			_, err := c.CreateVolume(ctx, &csi.CreateVolumeRequest{})
			return err
		}, statusCode(codes.InvalidArgument)),
		ExpectDo(func() error {
			_, err := c.CreateVolume(ctx, &csi.CreateVolumeRequest{
				Name: "v",
				VolumeCapabilities: []*csi.VolumeCapability{{
					AccessType: &csi.VolumeCapability_Block{Block: &csi.VolumeCapability_BlockVolume{}},
				}},
			})
			return err
		}, statusCode(codes.InvalidArgument)),
		ExpectDo(func() error {
			_, err := volumeFromID("bad")
			return err
		}, ErrorMatch(regexpMust("invalid id"))),
	)

	Then(t, "node publish/unpublish 参数错误",
		ExpectDo(func() error {
			_, err := n.NodePublishVolume(ctx, &csi.NodePublishVolumeRequest{})
			return err
		}, statusCode(codes.InvalidArgument)),
		ExpectDo(func() error {
			_, err := n.NodePublishVolume(ctx, &csi.NodePublishVolumeRequest{
				VolumeCapability: &csi.VolumeCapability{},
			})
			return err
		}, statusCode(codes.InvalidArgument)),
		ExpectDo(func() error {
			_, err := n.NodePublishVolume(ctx, &csi.NodePublishVolumeRequest{
				VolumeCapability: &csi.VolumeCapability{},
				VolumeId:         "vol",
			})
			return err
		}, statusCode(codes.InvalidArgument)),
		ExpectDo(func() error {
			_, err := n.NodePublishVolume(ctx, &csi.NodePublishVolumeRequest{
				VolumeCapability: &csi.VolumeCapability{},
				VolumeId:         "vol",
				TargetPath:       "/tmp/target",
			})
			return err
		}, statusCode(codes.InvalidArgument)),
		ExpectDo(func() error {
			_, err := n.NodeUnpublishVolume(ctx, &csi.NodeUnpublishVolumeRequest{})
			return err
		}, statusCode(codes.InvalidArgument)),
		ExpectDo(func() error {
			_, err := n.NodeUnpublishVolume(ctx, &csi.NodeUnpublishVolumeRequest{VolumeId: "vol"})
			return err
		}, statusCode(codes.InvalidArgument)),
	)
}

func TestUnimplemented(t *testing.T) {
	ctx := context.Background()
	c := &controllerServer{DriverContext: newFakeDriverContext(t), l: logr.Discard()}
	n := &nodeServer{DriverContext: newFakeDriverContext(t), l: logr.Discard()}

	Then(t, "未实现接口返回 Unimplemented",
		ExpectDo(func() error { _, err := c.GetSnapshot(ctx, nil); return err }, statusCode(codes.Unimplemented)),
		ExpectDo(func() error { _, err := c.ControllerModifyVolume(ctx, nil); return err }, statusCode(codes.Unimplemented)),
		ExpectDo(func() error { _, err := c.ControllerPublishVolume(ctx, nil); return err }, statusCode(codes.Unimplemented)),
		ExpectDo(func() error { _, err := c.ControllerUnpublishVolume(ctx, nil); return err }, statusCode(codes.Unimplemented)),
		ExpectDo(func() error { _, err := c.ValidateVolumeCapabilities(ctx, nil); return err }, statusCode(codes.Unimplemented)),
		ExpectDo(func() error { _, err := c.ControllerGetVolume(ctx, nil); return err }, statusCode(codes.Unimplemented)),
		ExpectDo(func() error { _, err := c.GetCapacity(ctx, nil); return err }, statusCode(codes.Unimplemented)),
		ExpectDo(func() error { _, err := c.ListVolumes(ctx, nil); return err }, statusCode(codes.Unimplemented)),
		ExpectDo(func() error { _, err := c.ControllerExpandVolume(ctx, nil); return err }, statusCode(codes.Unimplemented)),
		ExpectDo(func() error { _, err := c.CreateSnapshot(ctx, nil); return err }, statusCode(codes.Unimplemented)),
		ExpectDo(func() error { _, err := c.DeleteSnapshot(ctx, nil); return err }, statusCode(codes.Unimplemented)),
		ExpectDo(func() error { _, err := c.ListSnapshots(ctx, nil); return err }, statusCode(codes.Unimplemented)),
		ExpectDo(func() error { _, err := n.NodeStageVolume(ctx, nil); return err }, statusCode(codes.Unimplemented)),
		ExpectDo(func() error { _, err := n.NodeUnstageVolume(ctx, nil); return err }, statusCode(codes.Unimplemented)),
		ExpectDo(func() error { _, err := n.NodeExpandVolume(ctx, nil); return err }, statusCode(codes.Unimplemented)),
	)
}

func TestDriverAndUtilBranches(t *testing.T) {
	d := &Driver{NodeID: "node"}
	initErr := d.Init(logr.WithLogger(context.Background(), logr.Discard()))
	Then(t, "Driver 初始化注册服务",
		Expect(initErr, Equal[error](nil)),
		Expect(d.dctx.NodeID, Equal("node")),
		Expect(d.svc != nil, Equal(true)),
	)

	scheme, addr, err := ParseEndpoint("unix:///tmp/csi.sock")
	vol := MustValue(t, func() (*volume, error) {
		return volumeFromID("file#host#base#uuid")
	})
	Then(t, "解析 endpoint 和 volume id",
		Expect(err, Equal[error](nil)),
		Expect(scheme, Equal("unix")),
		Expect(addr, Equal("/tmp/csi.sock")),
		Expect(vol.uuid, Equal("uuid")),
	)
}

func TestDriverServeAndShutdownBranches(t *testing.T) {
	d := &Driver{}
	Then(t, "非法 endpoint 直接返回错误",
		ExpectDo(func() error {
			return d.Serve(context.Background())
		}, ErrorMatch(regexpMust("invalid endpoint"))),
	)

	initialized := &Driver{}
	Must(t, func() error {
		return initialized.Init(logr.WithLogger(context.Background(), logr.Discard()))
	})
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	Then(t, "Shutdown 在 context 取消时停止 server",
		ExpectDo(func() error {
			return initialized.Shutdown(ctx)
		}),
	)
}

func statusCode(code codes.Code) ValueChecker[error] {
	return Be(func(err error) error {
		if status.Code(err) != code {
			return status.Errorf(codes.Internal, "expected %s got %s", code, status.Code(err))
		}
		return nil
	})
}

func regexpMust(pattern string) *regexp.Regexp {
	return regexp.MustCompile(pattern)
}
