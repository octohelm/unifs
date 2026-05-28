package s3

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
)

func (fsys *fs) putObject(ctx context.Context, name string, body io.Reader, size int64, headers http.Header) error {
	resp, err := fsys.doPresignedRequest(ctx, fsys.s3Client, http.MethodPut, fsys.path(name), headers, body, size)
	if err != nil {
		return err
	}
	defer closeResponse(resp)

	switch resp.StatusCode {
	case http.StatusOK, http.StatusCreated, http.StatusNoContent:
		return nil
	default:
		data, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("status code: %s: %s", resp.Status, string(data))
	}
}

type s3Object struct {
	ctx context.Context
	fs  *fs

	key  string
	size int64

	offset int64
	body   io.ReadCloser
}

func (o *s3Object) Read(p []byte) (int, error) {
	if len(p) == 0 {
		return 0, nil
	}
	if o.offset >= o.size {
		return 0, io.EOF
	}
	if o.body == nil {
		if err := o.open(); err != nil {
			return 0, err
		}
	}

	n, err := o.body.Read(p)
	o.offset += int64(n)
	return n, err
}

func (o *s3Object) Seek(offset int64, whence int) (int64, error) {
	next := offset
	switch whence {
	case io.SeekStart:
	case io.SeekCurrent:
		next = o.offset + offset
	case io.SeekEnd:
		next = o.size + offset
	default:
		return -1, os.ErrInvalid
	}
	if next < 0 {
		return -1, os.ErrInvalid
	}
	if next == o.offset {
		return o.offset, nil
	}

	if o.body != nil {
		if err := o.body.Close(); err != nil {
			return -1, err
		}
		o.body = nil
	}

	o.offset = next
	return o.offset, nil
}

func (o *s3Object) Close() error {
	if o.body == nil {
		return nil
	}
	err := o.body.Close()
	o.body = nil
	return err
}

func (o *s3Object) open() error {
	headers := http.Header{}
	if o.offset > 0 {
		headers.Set("Range", fmt.Sprintf("bytes=%d-", o.offset))
	}

	resp, err := o.fs.doPresignedRequest(o.ctx, o.fs.s3Client, http.MethodGet, o.key, headers, nil, -1)
	if err != nil {
		return err
	}

	if o.offset == 0 && resp.StatusCode == http.StatusOK {
		o.body = resp.Body
		return nil
	}
	if o.offset > 0 && resp.StatusCode == http.StatusPartialContent {
		o.body = resp.Body
		return nil
	}

	defer closeResponse(resp)
	data, _ := io.ReadAll(resp.Body)
	return fmt.Errorf("status code: %s: %s", resp.Status, string(data))
}
