package webdav

import (
	"bytes"
	"context"
	"encoding/xml"
	"io"
	"os"
	"path"
	"strings"
	"testing"

	. "github.com/octohelm/x/testing/v2"

	"github.com/octohelm/unifs/pkg/filesystem"
	clientpkg "github.com/octohelm/unifs/pkg/filesystem/webdav/client"
)

func TestFSBranches(t *testing.T) {
	c := &fakeClient{}
	fsys := NewFS(c).(*fs)

	Then(t, "禁止删除根目录和 append 打开",
		ExpectDo(func() error {
			return fsys.RemoveAll(context.Background(), "/")
		}, ErrorIs(os.ErrPermission)),
		ExpectDo(func() error {
			_, err := fsys.OpenFile(context.Background(), "file.txt", os.O_APPEND, 0o644)
			return err
		}, ErrorIs(ErrNotSupported)),
	)

	Then(t, "同名 rename 直接成功",
		ExpectDo(func() error {
			return fsys.Rename(context.Background(), "same", "same")
		}),
		Expect(c.moved, Equal(false)),
	)
}

func TestOpenDirAndOpenFile(t *testing.T) {
	dirResp := decodeResponse(t, `<D:response xmlns:D="DAV:">
<D:href>/dir/</D:href>
<D:propstat><D:prop><D:resourcetype><D:collection/></D:resourcetype></D:prop><D:status>HTTP/1.1 200 OK</D:status></D:propstat>
</D:response>`)
	fileResp := decodeResponse(t, `<D:response xmlns:D="DAV:">
<D:href>/dir/file.txt</D:href>
<D:propstat>
<D:prop>
<D:resourcetype></D:resourcetype>
<D:getcontentlength>5</D:getcontentlength>
<D:getlastmodified>Thu, 01 Jan 1970 00:00:01 GMT</D:getlastmodified>
</D:prop>
<D:status>HTTP/1.1 200 OK</D:status>
</D:propstat>
</D:response>`)

	c := &fakeClient{}
	c.propFind = func(_ context.Context, name string, _ clientpkg.Depth, _ *clientpkg.PropFind) (*clientpkg.MultiStatus, error) {
		switch name {
		case ".", "/":
			return clientpkg.NewMultiStatus(dirResp), nil
		case "dir/":
			return clientpkg.NewMultiStatus(dirResp), nil
		case "dir":
			return clientpkg.NewMultiStatus(dirResp), nil
		case "newdir/":
			if c.createdDir == "newdir/" {
				return clientpkg.NewMultiStatus(decodeResponse(t, `<D:response xmlns:D="DAV:">
<D:href>/newdir/</D:href>
<D:propstat><D:prop><D:resourcetype><D:collection/></D:resourcetype></D:prop><D:status>HTTP/1.1 200 OK</D:status></D:propstat>
</D:response>`)), nil
			}
			return clientpkg.NewMultiStatus(clientpkg.NewErrorResponse(name, clientpkg.HTTPErrorf(404, "missing"))), nil
		case "dir/file.txt/":
			return clientpkg.NewMultiStatus(clientpkg.NewErrorResponse(name, clientpkg.HTTPErrorf(404, "missing"))), nil
		case "dir/file.txt":
			return clientpkg.NewMultiStatus(fileResp), nil
		case path.Dir("dir/new.txt"):
			return clientpkg.NewMultiStatus(dirResp), nil
		default:
			return clientpkg.NewMultiStatus(clientpkg.NewErrorResponse(name, clientpkg.HTTPErrorf(404, "missing"))), nil
		}
	}
	c.openWrite = func(_ context.Context, _ string) (io.WriteCloser, error) {
		return nopWriteCloser{Writer: io.Discard}, nil
	}
	c.open = func(_ context.Context, _ string) (clientpkg.File, error) {
		return &stubClientFile{ReadCloser: io.NopCloser(strings.NewReader("hello"))}, nil
	}
	c.mkcol = func(_ context.Context, name string) error {
		c.createdDir = name
		return nil
	}
	fsys := NewFS(c).(*fs)

	dir := MustValue(t, func() (filesystem.File, error) {
		return fsys.openDir(context.Background(), "dir/")
	}).(*file)
	defer dir.Close()
	createdDir := MustValue(t, func() (filesystem.File, error) {
		return fsys.openDir(context.Background(), "newdir/")
	}).(*file)
	defer createdDir.Close()
	writer := MustValue(t, func() (filesystem.File, error) {
		return fsys.openFile(context.Background(), "dir/new.txt", os.O_CREATE|os.O_WRONLY)
	})
	reader := MustValue(t, func() (filesystem.File, error) {
		return fsys.openFile(context.Background(), "dir/file.txt", os.O_RDONLY)
	})
	defer reader.Close()

	buf := make([]byte, 5)
	_, readErr := reader.Read(buf)
	Then(t, "目录和文件打开分支",
		Expect(dir.Name(), Equal("dir")),
		Expect(createdDir.Name(), Equal("newdir")),
		Expect(c.createdDir, Equal("newdir/")),
		Expect(writer != nil, Equal(true)),
		Expect(readErr, Equal[error](nil)),
		Expect(string(buf), Equal("hello")),
	)

	conflictErr := MustValue(t, func() (error, error) {
		_, err := fsys.openDir(context.Background(), "dir/file.txt")
		return err, nil
	})
	missingParentErr := MustValue(t, func() (error, error) {
		_, err := fsys.openFile(context.Background(), "missing/new.txt", os.O_CREATE|os.O_WRONLY)
		return err, nil
	})
	var pathErr *os.PathError
	Then(t, "openDir 文件名冲突和缺父目录返回错误",
		Expect(conflictErr, ErrorIs(os.ErrExist)),
		Expect(missingParentErr, ErrorAs(&pathErr), ErrorIs(os.ErrNotExist)),
	)
	Then(t, "缺父目录错误保留父路径",
		Expect(pathErr.Path, Equal("missing")),
	)
}

func TestFileBranches(t *testing.T) {
	f := &file{
		node: &node{
			root: &fs{c: &fakeClient{}},
			name: "file.txt",
		},
	}

	Then(t, "无 writer/file 时返回 invalid",
		ExpectDo(func() error {
			_, err := f.Write([]byte("x"))
			return err
		}, ErrorIs(os.ErrInvalid)),
		ExpectDo(func() error {
			_, err := f.Read(make([]byte, 1))
			return err
		}, ErrorIs(os.ErrInvalid)),
	)

	f.writer = nopWriteCloser{Writer: io.Discard}
	f.file = &stubClientFile{
		ReadCloser: io.NopCloser(strings.NewReader("abc")),
		seeker:     bytes.NewReader([]byte("abc")),
	}
	pos := MustValue(t, func() (int64, error) {
		return f.Seek(1, io.SeekStart)
	})
	buf := make([]byte, 3)
	_, readErr := f.Read(buf)
	_, writeErr := f.Write([]byte("x"))
	Then(t, "读写关闭委托到底层对象",
		Expect(f.Name(), Equal("file.txt")),
		Expect(pos, Equal(int64(1))),
		Expect(readErr, Equal[error](nil)),
		Expect(writeErr, Equal[error](nil)),
		ExpectDo(f.Close),
	)
}

func decodeResponse(t *testing.T, raw string) *clientpkg.Response {
	return MustValue(t, func() (*clientpkg.Response, error) {
		var resp clientpkg.Response
		if err := xml.Unmarshal([]byte(raw), &resp); err != nil {
			return nil, err
		}
		return &resp, nil
	})
}

type fakeClient struct {
	propFind   func(context.Context, string, clientpkg.Depth, *clientpkg.PropFind) (*clientpkg.MultiStatus, error)
	move       func(context.Context, string, string, bool) error
	delete     func(context.Context, string) error
	openWrite  func(context.Context, string) (io.WriteCloser, error)
	open       func(context.Context, string) (clientpkg.File, error)
	mkcol      func(context.Context, string) error
	moved      bool
	createdDir string
}

func (f *fakeClient) MkCol(ctx context.Context, name string) error {
	if f.mkcol != nil {
		return f.mkcol(ctx, name)
	}
	return nil
}

func (f *fakeClient) PropFind(ctx context.Context, name string, depth clientpkg.Depth, propfind *clientpkg.PropFind) (*clientpkg.MultiStatus, error) {
	if f.propFind != nil {
		return f.propFind(ctx, name, depth, propfind)
	}
	return nil, clientpkg.HTTPErrorf(404, "missing")
}

func (f *fakeClient) Move(ctx context.Context, src string, dest string, overwrite bool) error {
	f.moved = true
	if f.move != nil {
		return f.move(ctx, src, dest, overwrite)
	}
	return nil
}

func (f *fakeClient) Delete(ctx context.Context, name string) error {
	if f.delete != nil {
		return f.delete(ctx, name)
	}
	return nil
}

func (f *fakeClient) OpenWrite(ctx context.Context, name string) (io.WriteCloser, error) {
	if f.openWrite != nil {
		return f.openWrite(ctx, name)
	}
	return nopWriteCloser{Writer: io.Discard}, nil
}

func (f *fakeClient) Open(ctx context.Context, name string) (clientpkg.File, error) {
	if f.open != nil {
		return f.open(ctx, name)
	}
	return &stubClientFile{ReadCloser: io.NopCloser(strings.NewReader(""))}, nil
}

type nopWriteCloser struct{ io.Writer }

func (nopWriteCloser) Close() error { return nil }

type stubClientFile struct {
	io.ReadCloser
	seeker io.Seeker
}

func (s *stubClientFile) Seek(offset int64, whence int) (int64, error) {
	if s.seeker == nil {
		return 0, os.ErrInvalid
	}
	return s.seeker.Seek(offset, whence)
}
