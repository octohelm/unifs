package client

import (
	"io"
	"os"
	"sync"

	"golang.org/x/sync/errgroup"
)

type File interface {
	io.Seeker
	io.ReadCloser
}

var _ File = &file{}

type file struct {
	mu     sync.Mutex
	info   os.FileInfo
	pos    int64
	seeked bool

	lastBody  io.ReadCloser
	doRequest func(offset int64, end int64) (io.ReadCloser, error)
}

func (f *file) Seek(offset int64, whence int) (int64, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	npos := f.pos

	switch whence {
	case io.SeekStart:
		npos = offset
	case io.SeekCurrent:
		npos += offset
	case io.SeekEnd:
		npos = f.info.Size() + offset
	default:
		npos = -1
	}
	if npos < 0 {
		return 0, os.ErrInvalid
	}

	f.pos = npos
	f.seeked = true

	return f.pos, nil
}

func (f *file) Read(p []byte) (n int, err error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	if f.lastBody == nil {
		offset := int64(-1)
		if f.seeked {
			offset = f.pos
		}
		body, err := f.doRequest(offset, 0)
		if err != nil {
			return 0, err
		}
		f.lastBody = body
	}

	readBytes, _ := f.lastBody.Read(p)
	f.pos += int64(readBytes)

	if f.pos >= f.info.Size() {
		return readBytes, io.EOF
	}
	return readBytes, nil
}

func (f *file) Close() error {
	f.mu.Lock()
	defer f.mu.Unlock()

	eg := errgroup.Group{}

	eg.Go(func() error {
		return nil
	})

	eg.Go(func() error {
		if f.lastBody != nil {
			return f.lastBody.Close()
		}
		f.lastBody = nil
		return nil
	})

	return eg.Wait()
}
