package ftp

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"strings"
	"sync/atomic"
	"time"

	ftpserver "github.com/fclairamb/ftpserverlib"
	"github.com/spf13/afero"

	"github.com/innoai-tech/infra/pkg/configuration"
	"github.com/octohelm/x/logr"

	"github.com/octohelm/unifs/pkg/aferofsutil"
	"github.com/octohelm/unifs/pkg/filesystem"
	fslogr "github.com/octohelm/unifs/pkg/filesystem/logr"
)

var _ configuration.Server = &Server{}

type Server struct {
	Addr        string `flag:"addr,omitzero"`
	PublicHost  string `flag:"public-host,omitzero"`
	DisableMLST bool   `flag:"disable-mlst,omitzero"`
	DisableMLSD bool   `flag:"disable-mlsd,omitzero"`

	ftp *ftpserver.FtpServer
}

// SetDefaults 填充默认监听地址和公开主机。
func (s *Server) SetDefaults() {
	if s.Addr == "" {
		s.Addr = "0.0.0.0:2121"
	}

	if s.PublicHost == "" {
		s.PublicHost = strings.Split(s.Addr, ":")[0]
		if s.PublicHost == "" {
			s.PublicHost = "0.0.0.0"
		}
	}
}

// Serve 启动 FTP 服务。
func (s *Server) Serve(ctx context.Context) error {
	if s.ftp == nil {
		d := &driver{
			fs:     filesystem.Context.From(ctx),
			logger: logr.FromContext(ctx),
		}

		d.ListenAddr = s.Addr
		d.PublicHost = s.PublicHost
		d.DisableMLST = s.DisableMLST
		d.DisableMLSD = s.DisableMLSD

		s.ftp = ftpserver.NewFtpServer(d)

		logr.FromContext(ctx).Info(fmt.Sprintf("ftp serve on %s", s.Addr))

		if err := s.ftp.ListenAndServe(); err != nil {
			return fmt.Errorf("listen FTP server %q: %w", s.Addr, err)
		}
		return nil
	}
	return nil
}

// Shutdown 在 FTP 服务运行时停止它。
func (s *Server) Shutdown(ctx context.Context) error {
	if s.ftp != nil {
		if err := s.ftp.Stop(); err != nil {
			return fmt.Errorf("stop FTP server: %w", err)
		}
	}
	return nil
}

type driver struct {
	ftpserver.Settings

	logger logr.Logger

	fs filesystem.FileSystem

	nbClients       atomic.Int64
	zeroClientEvent chan error
}

func (s *driver) GetSettings() (*ftpserver.Settings, error) {
	return &s.Settings, nil
}

func (s *driver) AuthUser(cc ftpserver.ClientContext, user, pass string) (ftpserver.ClientDriver, error) {
	fs := aferofsutil.From(fslogr.Wrap(s.fs, s.logger.WithValues("ftp", "server")))

	s.logger.WithValues("user", user).Info("auth")

	return &ClientDriver{Fs: fs}, nil
}

func (s *driver) GetTLSConfig() (*tls.Config, error) {
	return nil, nil
}

type ClientDriver struct {
	afero.Fs
}

// ErrTimeout 表示优雅等待超过超时时间。
var ErrTimeout = errors.New("timeout")

func (s *driver) ClientConnected(cc ftpserver.ClientContext) (string, error) {
	s.nbClients.Add(1)

	s.logger.WithValues(
		"client.id", cc.ID(),
		"remote.addr", cc.RemoteAddr(),
		"path", cc.Path(),
	).Info("client connected")

	return "ftpserver", nil
}

// ClientDisconnected 在用户断开连接时调用，即使用户从未认证。
func (s *driver) ClientDisconnected(cc ftpserver.ClientContext) {
	s.nbClients.Add(-1)

	s.logger.WithValues(
		"client.id", cc.ID(),
		"remote.addr", cc.RemoteAddr(),
	).Info(
		"disconnected",
	)

	s.considerEnd()
}

func (s *driver) WaitGracefully(timeout time.Duration) error {
	s.logger.Info("waiting for last client to disconnect...")

	defer func() { s.zeroClientEvent = nil }()

	select {
	case err := <-s.zeroClientEvent:
		return err
	case <-time.After(timeout):
		return ErrTimeout
	}
}

func (s *driver) Stop() {
	s.zeroClientEvent = make(chan error, 1)
	s.considerEnd()
}

func (s *driver) considerEnd() {
	if s.nbClients.Load() == 0 && s.zeroClientEvent != nil {
		s.zeroClientEvent <- nil
		close(s.zeroClientEvent)
	}
}
