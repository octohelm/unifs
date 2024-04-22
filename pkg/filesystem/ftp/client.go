package ftp

import (
	"context"
	"io"
	"net/url"
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
	MaxConnections int32
	ConnectTimeout time.Duration

	count int64
}

func (p *Pool) Conn(ctx context.Context, args ...any) (Conn, error) {
	connectTimeout := time.Second * 5
	if p.ConnectTimeout > 0 {
		connectTimeout = p.ConnectTimeout
	}

	c, err := ftp.Dial(
		p.Addr,
		ftp.DialWithContext(ctx),
		ftp.DialWithTimeout(connectTimeout),
	)
	if err != nil {
		return nil, err
	}

	if p.Auth != nil {
		pass, _ := p.Auth.Password()
		if err := c.Login(p.Auth.Username(), pass); err != nil {
			return nil, err
		}
	} else {
		if err := c.Login("anonymous", "anonymous"); err != nil {
			return nil, err
		}
	}

	return &conn{conn: c}, nil
}

type conn struct {
	conn *ftp.ServerConn
}

func (c *conn) Close() error {
	return c.conn.Quit()
}

func (c *conn) MakeDir(path string) error {
	return c.conn.MakeDir(path)
}

func (c *conn) GetEntry(path string) (*ftp.Entry, error) {
	return c.conn.GetEntry(path)
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
	return c.conn.List(path)
}

func (c *conn) StorFrom(path string, reader io.Reader, offset uint64) error {
	return c.conn.StorFrom(path, reader, offset)
}

func (c *conn) RetrFrom(path string, offset uint64) (*ftp.Response, error) {
	return c.conn.RetrFrom(path, offset)
}
