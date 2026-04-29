package client

import (
	"encoding/xml"
	"errors"
	"net/http"
	"regexp"
	"testing"
	"time"

	. "github.com/octohelm/x/testing/v2"
)

func TestStatus(t *testing.T) {
	var status Status
	Must(t, func() error {
		return status.UnmarshalText([]byte("HTTP/1.1 404 Not Found"))
	})
	Then(t, "解析 HTTP status 文本",
		Expect(status, Equal(Status{Code: http.StatusNotFound, Text: "Not Found"})),
		Expect(status.Err(), ErrorAsType[*HTTPError]()),
	)

	text := MustValue(t, (&Status{Code: http.StatusOK}).MarshalText)
	Then(t, "格式化 HTTP status 文本",
		Expect(string(text), Equal("HTTP/1.1 200 OK")),
	)

	Then(t, "非法 status 文本返回错误",
		ExpectDo(func() error {
			return status.UnmarshalText([]byte("bad"))
		}, ErrorMatch(regexpMust("expected 3 fields"))),
	)
}

func TestDepthAndOverwrite(t *testing.T) {
	depth := MustValue(t, func() (Depth, error) {
		return ParseDepth("infinity")
	})

	Then(t, "解析和格式化 WebDAV header",
		Expect(depth, Equal(DepthInfinity)),
		Expect(DepthZero.String(), Equal("0")),
		Expect(DepthOne.String(), Equal("1")),
		Expect(DepthInfinity.String(), Equal("infinity")),
		Expect(MustValue(t, func() (bool, error) { return ParseOverwrite("T") }), Equal(true)),
		Expect(MustValue(t, func() (bool, error) { return ParseOverwrite("F") }), Equal(false)),
		Expect(FormatOverwrite(true), Equal("T")),
		Expect(FormatOverwrite(false), Equal("F")),
	)

	Then(t, "非法 header 返回错误",
		ExpectDo(func() error {
			_, err := ParseDepth("bad")
			return err
		}, ErrorMatch(regexpMust("invalid Depth"))),
		ExpectDo(func() error {
			_, err := ParseOverwrite("bad")
			return err
		}, ErrorMatch(regexpMust("invalid Overwrite"))),
	)
}

func TestHrefETagAndTime(t *testing.T) {
	var href Href
	Must(t, func() error {
		return href.UnmarshalText([]byte("/dir/file.txt"))
	})
	Then(t, "解析 href",
		Expect(href.String(), Equal("/dir/file.txt")),
	)

	var etag ETag
	Must(t, func() error {
		return etag.UnmarshalText([]byte(`"abc"`))
	})
	Then(t, "解析和格式化 ETag",
		Expect(etag.String(), Equal(`"abc"`)),
		Expect(string(MustValue(t, etag.MarshalText)), Equal(`"abc"`)),
	)

	var tm Time
	Must(t, func() error {
		return tm.UnmarshalText([]byte("Mon, 02 Jan 2006 15:04:05 GMT"))
	})
	Then(t, "解析和格式化 HTTP 时间",
		Expect(time.Time(tm), Equal(time.Date(2006, 1, 2, 15, 4, 5, 0, time.UTC))),
		Expect(string(MustValue(t, tm.MarshalText)), Equal("Mon, 02 Jan 2006 15:04:05 GMT")),
	)
}

func TestPropAndResponse(t *testing.T) {
	ms := NewMultiStatus(NewOKResponse("/ok"))
	Then(t, "构造 multistatus 和 ok response",
		Expect(len(ms.Responses), Equal(1)),
		Expect(ms.Responses[0].Status.Code, Equal(http.StatusOK)),
	)

	resp := decodeResponse(t, `<D:response xmlns:D="DAV:">
<D:href>/prefix/file.txt</D:href>
<D:propstat>
<D:prop>
<D:resourcetype></D:resourcetype>
<D:getcontentlength>12</D:getcontentlength>
<D:getlastmodified>Thu, 01 Jan 1970 00:02:03 GMT</D:getlastmodified>
</D:prop>
<D:status>HTTP/1.1 200 OK</D:status>
</D:propstat>
</D:response>`)
	resp.Prefix = "/prefix"

	info := MustValue(t, resp.FileInfo)
	pathname := MustValue(t, resp.Path)
	Then(t, "response 解码为文件信息",
		Expect(pathname, Equal("/file.txt")),
		Expect(info.Name(), Equal("file.txt")),
		Expect(info.Size(), Equal(int64(12))),
		Expect(info.IsDir(), Equal(false)),
	)

	dirResp := decodeResponse(t, `<D:response xmlns:D="DAV:">
<D:href>/dir</D:href>
<D:propstat>
<D:prop><D:resourcetype><D:collection/></D:resourcetype></D:prop>
<D:status>HTTP/1.1 200 OK</D:status>
</D:propstat>
</D:response>`)
	dirInfo := MustValue(t, dirResp.FileInfo)
	Then(t, "collection response 解码为目录信息",
		Expect(dirInfo.Name(), Equal("dir")),
		Expect(dirInfo.IsDir(), Equal(true)),
	)

	errResp := NewErrorResponse("/missing", HTTPErrorf(http.StatusNotFound, "missing"))
	Then(t, "error response 保留 HTTP 状态",
		Expect(errResp.Status.Code, Equal(http.StatusNotFound)),
		Expect(errResp.Err(), ErrorAsType[*HTTPError]()),
		Expect(IsNotFound(errResp.Err()), Equal(true)),
	)

	var encoded Response
	Must(t, func() error {
		return encoded.EncodeProp(http.StatusOK, &DisplayName{Name: "file.txt"})
	})
	var display DisplayName
	Then(t, "EncodeProp 后可 DecodeProp",
		Expect(encoded.PropStats[0].Status.Code, Equal(http.StatusOK)),
		Expect(len(encoded.PropStats[0].Prop.Raw), Equal(1)),
		Expect(display.Name, Equal("")),
	)
}

func TestRawXMLValue(t *testing.T) {
	var prop Prop
	Must(t, func() error {
		return xml.Unmarshal([]byte(`<D:prop xmlns:D="DAV:"><D:displayname>file.txt</D:displayname></D:prop>`), &prop)
	})

	var out DisplayName
	Must(t, func() error {
		return prop.Decode(&out)
	})
	Then(t, "raw XML value 可延迟解码",
		Expect(out.Name, Equal("file.txt")),
	)

	raw := NewRawXMLElement(xml.Name{Space: Namespace, Local: "displayname"}, nil, nil)
	name, ok := raw.XMLName()
	Then(t, "raw XML value 暴露元素名",
		Expect(name, Equal(DisplayNameName)),
		Expect(ok, Equal(true)),
	)
}

func TestHTTPError(t *testing.T) {
	cause := errors.New("boom")
	err := HTTPErrorf(http.StatusBadGateway, "upstream: %w", cause)

	Then(t, "HTTPError 支持错误链",
		Expect(err.Error(), Equal("502 Bad Gateway: upstream: boom")),
		ExpectDo(func() error {
			return err
		}, ErrorIs(cause)),
		Expect(HTTPErrorFromError(nil), Equal((*HTTPError)(nil))),
		Expect(HTTPErrorFromError(cause).Code, Equal(http.StatusInternalServerError)),
	)
}

func regexpMust(pattern string) *regexp.Regexp {
	return regexp.MustCompile(pattern)
}

func decodeResponse(t *testing.T, data string) *Response {
	return MustValue(t, func() (*Response, error) {
		var resp Response
		if err := xml.Unmarshal([]byte(data), &resp); err != nil {
			return nil, err
		}
		return &resp, nil
	})
}
