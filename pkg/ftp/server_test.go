package ftp

import (
	"context"
	"crypto/tls"
	"net"
	"testing"
	"time"

	ftpserver "github.com/fclairamb/ftpserverlib"

	"github.com/octohelm/x/logr"
	. "github.com/octohelm/x/testing/v2"

	"github.com/octohelm/unifs/pkg/filesystem"
)

func TestServerDefaultsShutdown(t *testing.T) {
	s := &Server{}
	s.SetDefaults()
	Then(t, "填充 FTP 默认配置",
		Expect(s.Addr, Equal("0.0.0.0:2121")),
		Expect(s.PublicHost, Equal("0.0.0.0")),
		ExpectDo(func() error {
			return s.Shutdown(context.Background())
		}),
	)
}

func TestDriver(t *testing.T) {
	d := &driver{
		logger:          logr.Discard(),
		fs:              filesystem.NewMemFS(),
		zeroClientEvent: make(chan error, 1),
	}

	settings, err := d.GetSettings()
	client, authErr := d.AuthUser(fakeClientContext{}, "user", "pass")
	tlsConfig, tlsErr := d.GetTLSConfig()
	name, connectedErr := d.ClientConnected(fakeClientContext{})
	d.ClientDisconnected(fakeClientContext{})

	Then(t, "driver 基本回调",
		Expect(err, Equal[error](nil)),
		Expect(settings, Equal(&d.Settings)),
		Expect(authErr, Equal[error](nil)),
		Expect(client != nil, Equal(true)),
		Expect(tlsErr, Equal[error](nil)),
		Expect(tlsConfig, Equal((*tls.Config)(nil))),
		Expect(connectedErr, Equal[error](nil)),
		Expect(name, Equal("ftpserver")),
	)

	d.Stop()
	Then(t, "无客户端时优雅等待立即结束",
		ExpectDo(func() error {
			return d.WaitGracefully(time.Second)
		}),
	)

	d.nbClients.Add(1)
	d.Stop()
	Then(t, "仍有客户端时等待超时",
		ExpectDo(func() error {
			return d.WaitGracefully(time.Millisecond)
		}, ErrorIs(ErrTimeout)),
	)
}

type fakeClientContext struct{}

func (fakeClientContext) ID() uint32                                       { return 1 }
func (fakeClientContext) RemoteAddr() net.Addr                             { return fakeAddr("remote") }
func (fakeClientContext) LocalAddr() net.Addr                              { return fakeAddr("local") }
func (fakeClientContext) Path() string                                     { return "/" }
func (fakeClientContext) SetPath(string)                                   {}
func (fakeClientContext) SetListPath(string)                               {}
func (fakeClientContext) SetDebug(bool)                                    {}
func (fakeClientContext) Debug() bool                                      { return false }
func (fakeClientContext) GetClientVersion() string                         { return "test" }
func (fakeClientContext) Close() error                                     { return nil }
func (fakeClientContext) HasTLSForControl() bool                           { return false }
func (fakeClientContext) HasTLSForTransfers() bool                         { return false }
func (fakeClientContext) GetLastCommand() string                           { return "" }
func (fakeClientContext) GetLastDataChannel() ftpserver.DataChannel        { return 0 }
func (fakeClientContext) SetTLSRequirement(ftpserver.TLSRequirement) error { return nil }
func (fakeClientContext) SetExtra(any)                                     {}
func (fakeClientContext) Extra() any                                       { return nil }

var _ ftpserver.ClientContext = fakeClientContext{}

type fakeAddr string

func (a fakeAddr) Network() string { return string(a) }
func (a fakeAddr) String() string  { return string(a) }
