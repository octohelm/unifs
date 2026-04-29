package ftp

import (
	"context"
	"fmt"
	"io"
	"os"
	"sync"
	"time"

	"github.com/jlaffaye/ftp"
	"golang.org/x/sync/errgroup"
)

type file struct {
	entry *ftp.Entry

	ctx    context.Context
	client Client
	flag   int
	offset uint64

	readCloser  io.ReadCloser
	writeCloser io.WriteCloser
	err         error

	once sync.Once
}

func (f *file) Close() error {
	eg := &errgroup.Group{}

	if f.writeCloser != nil {
		eg.Go(func() error {
			err := f.writeCloser.Close()
			f.writeCloser = nil
			return err
		})
	}

	if f.readCloser != nil {
		eg.Go(func() error {
			err := f.readCloser.Close()
			f.readCloser = nil
			return err
		})
	}

	if err := eg.Wait(); err != nil {
		return normalizeError("close", f.entry.Name, err)
	}
	return nil
}

func (f *file) Seek(offset int64, whence int) (int64, error) {
	switch whence {
	case 0:
		f.offset = uint64(offset)
	case 1:
		f.offset += uint64(offset)
	case 2:
		if f.entry.Size == 0 {
			return -1, normalizeError("seek", f.entry.Name, os.ErrInvalid)
		}
		f.offset = f.entry.Size - uint64(offset)
	}

	return int64(f.offset), nil
}

func (f *file) Read(p []byte) (n int, err error) {
	if f.err != nil {
		return 0, f.err
	}

	f.once.Do(func() {
		conn, err := f.client.Conn(f.ctx)
		if err != nil {
			f.err = normalizeError("read", f.entry.Name, err)
			return
		}

		resp, err := conn.RetrFrom(f.entry.Name, f.offset)
		if err != nil {
			f.err = normalizeError("read", f.entry.Name, err)
			return
		}

		f.readCloser = &readCloser{Response: resp, conn: conn}
	})

	if f.err != nil {
		return 0, f.err
	}

	n, err = f.readCloser.Read(p)
	if err != nil && err != io.EOF {
		return n, normalizeError("read", f.entry.Name, err)
	}
	return n, err
}

type readCloser struct {
	*ftp.Response
	conn Conn
}

func (c *readCloser) Close() error {
	eg := &errgroup.Group{}
	eg.Go(func() error {
		if err := c.Response.Close(); err != nil {
			return fmt.Errorf("close FTP response: %w", err)
		}
		return nil
	})
	eg.Go(func() error {
		if err := c.conn.Close(); err != nil {
			return fmt.Errorf("close FTP connection: %w", err)
		}
		return nil
	})
	return eg.Wait()
}

func (f *file) Write(p []byte) (n int, err error) {
	if !(f.flag&os.O_WRONLY != 0 || f.flag&os.O_RDWR != 0) {
		return 0, normalizeError("write", f.entry.Name, os.ErrPermission)
	}

	f.once.Do(func() {
		conn, err := f.client.Conn(f.ctx, "write", f.entry.Name)
		if err != nil {
			f.err = normalizeError("write", f.entry.Name, err)
			return
		}

		r, w := io.Pipe()

		ww := &writeCloser{WriteCloser: w}
		ww.wg.Add(1)

		go func() {
			defer func() {
				_ = conn.Close()
				_ = r.Close()
				ww.wg.Done()
			}()

			if err := conn.StorFrom(f.entry.Name, r, f.offset); err != nil {
				f.err = normalizeError("write", f.entry.Name, err)
			}
		}()

		f.writeCloser = ww
	})

	if f.err != nil {
		return 0, f.err
	}

	n, err = f.writeCloser.Write(p)
	if err != nil {
		return n, normalizeError("write", f.entry.Name, err)
	}
	return n, nil
}

type writeCloser struct {
	wg sync.WaitGroup
	io.WriteCloser
}

func (c *writeCloser) Close() error {
	err := c.WriteCloser.Close()
	c.wg.Wait()
	if err != nil {
		return fmt.Errorf("close FTP write pipe: %w", err)
	}
	return nil
}

func (f *file) Readdir(count int) ([]os.FileInfo, error) {
	conn, err := f.client.Conn(f.ctx)
	if err != nil {
		return nil, normalizeError("readdir", f.entry.Name, err)
	}
	defer conn.Close()

	entries, err := conn.List(f.entry.Name)
	if err != nil {
		return nil, normalizeError("readdir", f.entry.Name, err)
	}

	n := len(entries)
	if count > 0 {
		n = 0
	}

	list := make([]os.FileInfo, n)

	for i := 0; i < n; i++ {
		list[i] = &entry{
			name:  entries[i].Name,
			entry: entries[i],
		}
	}

	return list, nil
}

func (f *file) Stat() (os.FileInfo, error) {
	return &entry{
		name:  f.entry.Name,
		entry: f.entry,
	}, nil
}

type entry struct {
	name  string
	entry *ftp.Entry
}

func (f *entry) Name() string {
	return f.name
}

func (f *entry) Size() int64 {
	return int64(f.entry.Size)
}

func (f *entry) Mode() os.FileMode {
	if f.IsDir() {
		return os.ModeDir
	}
	return os.ModePerm
}

func (f *entry) ModTime() time.Time {
	return f.entry.Time
}

func (f *entry) IsDir() bool {
	return f.entry.Type == ftp.EntryTypeFolder
}

func (f *entry) Sys() any {
	return nil
}
