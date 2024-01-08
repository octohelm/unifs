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
	"github.com/go-courier/logr"
	"github.com/innoai-tech/infra/pkg/configuration"
	"github.com/spf13/afero"

	"github.com/octohelm/unifs/pkg/aferofsutil"
	"github.com/octohelm/unifs/pkg/filesystem"
	fslogr "github.com/octohelm/unifs/pkg/filesystem/logr"
)

var _ configuration.Server = &Server{}

type Server struct {
	Addr       string `flag:"addr,omitempty"`
	PublicHost string `json:"publicHost,omitempty"`

	ftp *ftpserver.FtpServer
}

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

func (s *Server) Serve(ctx context.Context) error {
	if s.ftp == nil {
		s.ftp = ftpserver.NewFtpServer(&driver{
			ListenAddr: s.Addr,
			PublicHost: s.PublicHost,
			fs:         filesystem.Context.From(ctx),
			logger:     logr.FromContext(ctx),
		})

		logr.FromContext(ctx).Info(fmt.Sprintf("ftp serve on %s", s.Addr))

		return s.ftp.ListenAndServe()
	}
	return nil
}

func (s *Server) Shutdown(ctx context.Context) error {
	if s.ftp != nil {
		return s.ftp.Stop()
	}
	return nil
}

type driver struct {
	ListenAddr string
	PublicHost string

	logger logr.Logger

	fs filesystem.FileSystem

	nbClients       int64
	zeroClientEvent chan error
}

func (s *driver) GetSettings() (*ftpserver.Settings, error) {
	return &ftpserver.Settings{
		ListenAddr: s.ListenAddr,
		PublicHost: s.PublicHost,
	}, nil
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

var ErrTimeout = errors.New("timeout")

func (s *driver) ClientConnected(cc ftpserver.ClientContext) (string, error) {
	atomic.AddInt64(&s.nbClients, 1)

	s.logger.WithValues(
		"client.id", cc.ID(),
		"remote.addr", cc.RemoteAddr(),
		"path", cc.Path(),
	).Info("client connected")

	return "ftpserver", nil
}

// ClientDisconnected is called when the user disconnects, even if he never authenticated
func (s *driver) ClientDisconnected(cc ftpserver.ClientContext) {
	atomic.AddInt64(&s.nbClients, -1)

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
	if s.nbClients == 0 && s.zeroClientEvent != nil {
		s.zeroClientEvent <- nil
		close(s.zeroClientEvent)
	}
}
