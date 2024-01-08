package ftp

import (
	"context"
	"github.com/jackc/puddle/v2"
	"io"
	"net/url"
	"sync"
	"sync/atomic"
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
	Attr           string
	Auth           *url.Userinfo
	MaxConnections int32
	ConnectTimeout time.Duration

	p     *puddle.Pool[*ftp.ServerConn]
	err   error
	once  sync.Once
	count int64
}

func (p *Pool) Conn(ctx context.Context, args ...any) (Conn, error) {
	p.once.Do(func() {
		maxConnections := int32(10)
		if p.MaxConnections > 0 {
			maxConnections = p.MaxConnections
		}

		pool, err := puddle.NewPool(&puddle.Config[*ftp.ServerConn]{
			Constructor: func(ctx context.Context) (res *ftp.ServerConn, err error) {
				c, err := ftp.Dial(
					p.Attr,
					ftp.DialWithContext(ctx),
					ftp.DialWithTimeout(p.ConnectTimeout),
				)

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

				if err != nil {
					return nil, err
				}

				return c, nil
			},
			Destructor: func(sc *ftp.ServerConn) {
				_ = sc.Quit()
			},
			MaxSize: maxConnections,
		})
		if err != nil {
			p.err = err
			return
		}
		p.p = pool
	})

	if p.err != nil {
		return nil, p.err
	}

	res, err := p.p.Acquire(ctx)
	if err != nil {
		return nil, err
	}

	idx := atomic.AddInt64(&p.count, 1)

	return &conn{
		idx: idx,
		res: res,
	}, nil
}

type conn struct {
	idx int64
	res *puddle.Resource[*ftp.ServerConn]
}

func (c *conn) Close() error {
	c.res.Release()
	return nil
}

func (c *conn) MakeDir(path string) error {
	return c.res.Value().MakeDir(path)
}

func (c *conn) GetEntry(path string) (*ftp.Entry, error) {
	return c.res.Value().GetEntry(path)
}

func (c *conn) Delete(name string) error {
	return c.res.Value().Delete(name)
}

func (c *conn) RemoveDirRecur(name string) error {
	return c.res.Value().RemoveDirRecur(name)
}

func (c *conn) Rename(oldName, newName string) error {
	return c.res.Value().Rename(oldName, newName)
}

func (c *conn) List(path string) ([]*ftp.Entry, error) {
	return c.res.Value().List(path)
}

func (c *conn) StorFrom(path string, reader io.Reader, offset uint64) error {
	return c.res.Value().StorFrom(path, reader, offset)
}

func (c *conn) RetrFrom(path string, offset uint64) (*ftp.Response, error) {
	return c.res.Value().RetrFrom(path, offset)
}
