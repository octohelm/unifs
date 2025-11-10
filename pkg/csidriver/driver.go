package csidriver

import (
	"context"
	"fmt"
	"net"
	"strings"

	"github.com/container-storage-interface/spec/lib/go/csi"
	"github.com/innoai-tech/infra/pkg/configuration"
	"github.com/octohelm/x/logr"
	"google.golang.org/grpc"

	"github.com/octohelm/unifs/internal/version"
	"github.com/octohelm/unifs/pkg/strfmt"
)

const (
	DefaultDriverName = "csi-driver.unifs.octohelm.tech"
	backend           = "backend"
)

var _ configuration.Server = &Driver{}

type Driver struct {
	Endpoint string `flag:"endpoint"`
	NodeID   string `flag:"nodeid"`

	dctx DriverContext
	svc  *grpc.Server
}

func (d *Driver) Init(ctx context.Context) error {
	logErr := func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		resp, err := handler(ctx, req)
		if err != nil {
			logr.FromContext(ctx).Error(err)
		}
		return resp, err
	}

	opts := []grpc.ServerOption{
		grpc.UnaryInterceptor(logErr),
	}

	d.dctx = DriverContext{
		Name:          DefaultDriverName,
		VendorVersion: version.Version(),
		NodeID:        d.NodeID,
	}

	d.dctx.AddControllerServiceCapabilities(csi.ControllerServiceCapability_RPC_CREATE_DELETE_VOLUME)

	d.dctx.AddVolumeCapabilityAccessModes(csi.VolumeCapability_AccessMode_SINGLE_NODE_WRITER)

	d.svc = grpc.NewServer(opts...)

	csi.RegisterIdentityServer(d.svc, &identityServer{
		DriverContext: d.dctx,
		l:             logr.FromContext(ctx).WithValues("server", "identity"),
	})

	csi.RegisterControllerServer(d.svc, &controllerServer{
		DriverContext: d.dctx,
		l:             logr.FromContext(ctx).WithValues("server", "controller"),
	})

	csi.RegisterNodeServer(d.svc, &nodeServer{
		DriverContext: d.dctx,
		l:             logr.FromContext(ctx).WithValues("server", "node"),
	})

	return nil
}

func (d *Driver) Serve(ctx context.Context) error {
	scheme, addr, err := ParseEndpoint(d.Endpoint)
	if err != nil {
		return err
	}

	listener, err := net.Listen(scheme, addr)
	if err != nil {
		return err
	}

	logr.FromContext(ctx).Info(fmt.Sprintf("Listening for connections on address: %s", listener.Addr()))

	return d.svc.Serve(listener)
}

func (d *Driver) Shutdown(ctx context.Context) error {
	done := make(chan error)

	go func() {
		d.svc.GracefulStop()
		done <- nil
	}()

	select {
	case <-ctx.Done():
		d.svc.Stop()
	case <-done:
		return nil
	}

	return nil
}

type DriverContext struct {
	Name          string
	VendorVersion string
	NodeID        string

	ControllerServiceCapabilities []*csi.ControllerServiceCapability
	VolumeCapabilityAccessModes   []*csi.VolumeCapability_AccessMode

	ns *nodeServer
}

func (n *DriverContext) AddControllerServiceCapabilities(cl ...csi.ControllerServiceCapability_RPC_Type) {
	var csc []*csi.ControllerServiceCapability
	for _, c := range cl {
		csc = append(csc, NewControllerServiceCapability(c))
	}
	n.ControllerServiceCapabilities = csc
}

func (n *DriverContext) AddVolumeCapabilityAccessModes(nl ...csi.VolumeCapability_AccessMode_Mode) {
	var nac []*csi.VolumeCapability_AccessMode
	for _, n := range nl {
		nac = append(nac, NewVolumeCapabilityAccessMode(n))
	}
	n.VolumeCapabilityAccessModes = nac
}

func newVolume(name string, size int64, secrets map[string]string, params map[string]string) (*volume, error) {
	vol := &volume{}
	vol.size = size

	vol.uuid = name

	for k, v := range secrets {
		switch k {
		case backend:
			e, err := strfmt.ParseEndpoint(v)
			if err != nil {
				return nil, err
			}

			vol.scheme = e.Scheme
			vol.host = e.Host()
			vol.base = strings.Trim(e.Path, "/")
		}
	}

	vol.id = strings.Join([]string{
		vol.scheme,
		vol.host,
		vol.base,
		vol.uuid,
	}, "#")

	return vol, nil
}

type volume struct {
	id     string
	uuid   string
	scheme string
	host   string
	base   string
	size   int64
}

func volumeFromID(id string) (*volume, error) {
	segments := strings.Split(id, "#")

	if len(segments) != 4 {
		return nil, fmt.Errorf("invalid id %s", id)
	}

	return &volume{
		id:     id,
		scheme: segments[0],
		host:   segments[1],
		base:   segments[2],
		uuid:   segments[3],
	}, nil
}
