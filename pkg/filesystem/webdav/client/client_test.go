package client

import (
	"bytes"
	"context"
	"encoding/xml"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"testing"
	"time"

	. "github.com/octohelm/x/testing/v2"

	"github.com/octohelm/unifs/pkg/filesystem/fsutil"
)

func TestClientRequests(t *testing.T) {
	var methods []string
	svc := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		methods = append(methods, r.Method)
		switch r.Method {
		case "PROPFIND":
			w.WriteHeader(http.StatusMultiStatus)
			_, _ = w.Write([]byte(`<D:multistatus xmlns:D="DAV:">
<D:response>
<D:href>/base/file.txt</D:href>
<D:propstat>
<D:prop>
<D:resourcetype></D:resourcetype>
<D:getcontentlength>5</D:getcontentlength>
<D:getlastmodified>Thu, 01 Jan 1970 00:00:01 GMT</D:getlastmodified>
</D:prop>
<D:status>HTTP/1.1 200 OK</D:status>
</D:propstat>
</D:response>
</D:multistatus>`))
		case "GET":
			if r.Header.Get("Range") == "" {
				_, _ = w.Write([]byte("hello"))
				return
			}
			_, _ = w.Write([]byte("llo"))
		case "PUT":
			_, _ = io.Copy(io.Discard, r.Body)
			w.WriteHeader(http.StatusCreated)
		case "MOVE", "DELETE", "MKCOL":
			w.WriteHeader(http.StatusNoContent)
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	}))
	defer svc.Close()

	c := MustValue(t, func() (Client, error) {
		return NewClient(svc.URL + "/base")
	})
	ctx := context.Background()

	Then(t, "执行 WebDAV 写操作",
		ExpectMust(func() error {
			return c.MkCol(ctx, "dir")
		}),
		ExpectMust(func() error {
			return c.Move(ctx, "file.txt", "moved.txt", true)
		}),
		ExpectMust(func() error {
			return c.Delete(ctx, "moved.txt")
		}),
	)

	writer := MustValue(t, func() (io.WriteCloser, error) {
		return c.OpenWrite(ctx, "file.txt")
	})
	_, err := writer.Write([]byte("hello"))
	Must(t, writer.Close)
	Then(t, "OpenWrite 上传内容",
		Expect(err, Equal[error](nil)),
	)

	f := MustValue(t, func() (File, error) {
		return c.Open(ctx, "file.txt")
	})
	_, err = f.Seek(2, io.SeekStart)
	buf := make([]byte, 3)
	n, readErr := f.Read(buf)
	Must(t, f.Close)
	Then(t, "Open 读取远端文件",
		Expect(err, Equal[error](nil)),
		Expect(n, Equal(3)),
		Expect(readErr, Equal(io.EOF)),
		Expect(string(buf), Equal("llo")),
		Expect(methods, Equal([]string{"MKCOL", "MOVE", "DELETE", "PUT", "PROPFIND", "GET"})),
	)
}

func TestClientHelpers(t *testing.T) {
	c := MustValue(t, func() (Client, error) {
		return NewClient("http://example.com/base")
	}).(*client)

	href := MustValue(t, func() (*url.URL, error) {
		return c.ResolveHref("/dir/file.txt")
	})
	header := http.Header{}
	setRange(header, 2, 0)
	lastBytesHeader := http.Header{}
	setRange(lastBytesHeader, 0, -5)
	rangeHeader := http.Header{}
	setRange(rangeHeader, 1, 3)
	ignoredHeader := http.Header{}
	setRange(ignoredHeader, 5, 3)

	req := MustValue(t, func() (*http.Request, error) {
		return c.reqXML(context.Background(), "PROPFIND", "file.txt", FileInfoPropFind)
	})

	Then(t, "构造请求辅助方法",
		Expect(href.String(), Equal("http://example.com/base/dir/file.txt")),
		Expect(header.Get("Range"), Equal("bytes=2-")),
		Expect(lastBytesHeader.Get("Range"), Equal("bytes=-5")),
		Expect(rangeHeader.Get("Range"), Equal("bytes=1-3")),
		Expect(ignoredHeader.Get("Range"), Equal("")),
		Expect(req.Header.Get("Content-Type"), Equal("text/xml; charset=utf-8")),
	)
}

func TestClientHTTPErrorResponses(t *testing.T) {
	svc := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case "DELETE":
			w.WriteHeader(http.StatusNotFound)
		case "MKCOL":
			w.WriteHeader(http.StatusInternalServerError)
		case "PROPFIND":
			w.WriteHeader(http.StatusNotFound)
		default:
			w.WriteHeader(http.StatusOK)
		}
	}))
	defer svc.Close()

	c := MustValue(t, func() (Client, error) {
		return NewClient(svc.URL)
	})
	cc := c.(*client)
	ctx := context.Background()
	ms := MustValue(t, func() (*MultiStatus, error) {
		return cc.PropFind(ctx, "missing", DepthZero, nil)
	})

	Then(t, "DELETE 404 视为成功，其他错误保留",
		ExpectMust(func() error {
			return c.Delete(ctx, "missing")
		}),
		ExpectDo(func() error {
			return c.MkCol(ctx, "dir")
		}, ErrorAsType[*HTTPError]()),
		Expect(len(ms.Responses), Equal(1)),
		Expect(ms.Responses[0].Status.Code, Equal(http.StatusNotFound)),
	)
}

func TestClientFile(t *testing.T) {
	f := &file{
		info: fsutil.NewFileInfo("file.txt", 5, time.Unix(0, 0)),
		doRequest: func(offset int64, end int64) (io.ReadCloser, error) {
			if offset == 2 {
				return io.NopCloser(bytes.NewReader([]byte("llo"))), nil
			}
			return io.NopCloser(bytes.NewReader([]byte("hello"))), nil
		},
	}

	pos := MustValue(t, func() (int64, error) {
		return f.Seek(2, io.SeekStart)
	})
	buf := make([]byte, 3)
	n, err := f.Read(buf)
	Must(t, f.Close)

	Then(t, "file 支持 seek 后读取",
		Expect(pos, Equal(int64(2))),
		Expect(n, Equal(3)),
		Expect(err, Equal(io.EOF)),
		Expect(string(buf), Equal("llo")),
	)

	Then(t, "非法 seek 返回错误",
		ExpectDo(func() error {
			_, err := f.Seek(-1, io.SeekStart)
			return err
		}, ErrorIs(os.ErrInvalid)),
	)
}

func TestRawXMLMarshal(t *testing.T) {
	prop := MustValue(t, func() (*Prop, error) {
		return EncodeProp(&DisplayName{Name: "file.txt"})
	})
	data := MustValue(t, func() ([]byte, error) {
		return xml.Marshal(&prop.Raw[0])
	})

	Then(t, "marshal-only RawXMLValue 可输出 XML",
		Expect(string(data), Equal(`<displayname xmlns="DAV:">file.txt</displayname>`)),
	)
}
