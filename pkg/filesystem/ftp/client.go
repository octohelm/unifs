package ftp

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"net/textproto"
	"net/url"
	"os"
	"path"
	"time"

	"github.com/jlaffaye/ftp"
)

type Client interface {
	Conn(ctx context.Context, args ...any) (Conn, error)
}

type Conn interface {
	Close() error

	MakeDir(name string) error
	RemoveDirRecur(name string) error

	Delete(name string) error
	Rename(oldName, newName string) error

	GetEntry(name string) (*ftp.Entry, error)
	List(path string) ([]*ftp.Entry, error)

	RetrFrom(path string, offset uint64) (*ftp.Response, error)
	StorFrom(path string, reader io.Reader, offset uint64) error
}

type Pool struct {
	Addr           string
	Auth           *url.Userinfo
	ConnectTimeout time.Duration

	EnableDebug bool
	ExplicitTLS bool
	TLSConfig   *tls.Config

	count int64
}

func (p *Pool) Conn(ctx context.Context, args ...any) (Conn, error) {
	connectTimeout := time.Second * 5
	if p.ConnectTimeout > 0 {
		connectTimeout = p.ConnectTimeout
	}

	options := []ftp.DialOption{
		ftp.DialWithContext(ctx),
		ftp.DialWithTimeout(connectTimeout),
	}

	if p.EnableDebug {
		options = append(options,
			ftp.DialWithDebugOutput(os.Stdout),
		)
	}

	if p.TLSConfig != nil {
		if p.ExplicitTLS {
			options = append(options,
				ftp.DialWithExplicitTLS(p.TLSConfig),
			)
		} else {
			options = append(options,
				ftp.DialWithTLS(p.TLSConfig),
			)
		}
	}

	c, err := ftp.Dial(p.Addr, options...)
	if err != nil {
		return nil, fmt.Errorf("ftp dial %q: %w", p.Addr, err)
	}

	if p.Auth != nil {
		pass, _ := p.Auth.Password()
		if err := c.Login(p.Auth.Username(), pass); err != nil {
			_ = c.Quit()
			return nil, fmt.Errorf("ftp login %q as %q: %w", p.Addr, p.Auth.Username(), err)
		}
	} else {
		if err := c.Login("anonymous", "anonymous"); err != nil {
			_ = c.Quit()
			return nil, fmt.Errorf("ftp anonymous login %q: %w", p.Addr, err)
		}
	}

	return &conn{
		conn: c,
	}, nil
}

type conn struct {
	conn *ftp.ServerConn
}

func (c *conn) Close() error {
	return c.conn.Quit()
}

func (c *conn) MakeDir(pathname string) error {
	return c.conn.MakeDir(pathname)
}

func (c *conn) GetEntry(pathname string) (*ftp.Entry, error) {
	e, err := c.conn.GetEntry(pathname)
	if err != nil {
		// 兼容 FTP 服务端不支持 MLST 的情况。
		terr := &textproto.Error{}
		if errors.As(err, &terr) {
			if terr.Code == ftp.StatusNotImplemented {
				if pathname == "" || pathname == "." || pathname == "/" {
					if _, err := c.List(pathname); err != nil {
						return nil, fmt.Errorf("ftp list %q: %w", pathname, err)
					}
					return &ftp.Entry{
						Type: ftp.EntryTypeFolder,
					}, nil
				}

				dir, base := path.Split(pathname)
				list, err := c.List(path.Clean(dir))
				if err != nil {
					return nil, fmt.Errorf("ftp list parent %q: %w", path.Clean(dir), err)
				}

				for _, x := range list {
					if x.Name == base {
						return x, nil
					}
				}

				return nil, &textproto.Error{
					Code: ftp.StatusFileUnavailable,
				}
			}
		}

		return nil, fmt.Errorf("ftp get entry %q: %w", pathname, err)
	}
	return e, nil
}

func (c *conn) Delete(name string) error {
	return c.conn.Delete(name)
}

func (c *conn) RemoveDirRecur(name string) error {
	return c.conn.RemoveDirRecur(name)
}

func (c *conn) Rename(oldName, newName string) error {
	return c.conn.Rename(oldName, newName)
}

func (c *conn) List(path string) ([]*ftp.Entry, error) {
	if path == "." {
		return c.conn.List("")
	}
	return c.conn.List(path)
}

func (c *conn) StorFrom(path string, reader io.Reader, offset uint64) error {
	return c.conn.StorFrom(path, reader, offset)
}

func (c *conn) RetrFrom(path string, offset uint64) (*ftp.Response, error) {
	return c.conn.RetrFrom(path, offset)
}
