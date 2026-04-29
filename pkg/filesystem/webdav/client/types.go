package client

import (
	"encoding/xml"
	"fmt"
	"net/http"
	"strconv"
	"time"
)

const Namespace = "DAV:"

var (
	ResourceTypeName     = xml.Name{Space: Namespace, Local: "resourcetype"}
	DisplayNameName      = xml.Name{Space: Namespace, Local: "displayname"}
	GetContentLengthName = xml.Name{Space: Namespace, Local: "getcontentlength"}
	GetContentTypeName   = xml.Name{Space: Namespace, Local: "getcontenttype"}
	GetLastModifiedName  = xml.Name{Space: Namespace, Local: "getlastmodified"}
	GetETagName          = xml.Name{Space: Namespace, Local: "getetag"}

	CurrentUserPrincipalName = xml.Name{Space: Namespace, Local: "current-user-principal"}
)

// https://tools.ietf.org/html/rfc4918#section-14.9
type Location struct {
	XMLName xml.Name `xml:"DAV: location"`
	Href    Href     `xml:"href"`
}

// https://tools.ietf.org/html/rfc4918#section-14.22
type PropStat struct {
	XMLName             xml.Name `xml:"DAV: propstat"`
	Prop                Prop     `xml:"prop"`
	Status              Status   `xml:"status"`
	ResponseDescription string   `xml:"responsedescription,omitzero"`
	Error               *Error   `xml:"error,omitzero"`
}

// https://tools.ietf.org/html/rfc4918#section-14.18
type Prop struct {
	XMLName xml.Name      `xml:"DAV: prop"`
	Raw     []RawXMLValue `xml:",any"`
}

func EncodeProp(values ...any) (*Prop, error) {
	l := make([]RawXMLValue, len(values))
	for i, v := range values {
		raw, err := EncodeRawXMLElement(v)
		if err != nil {
			return nil, fmt.Errorf("encode property %T: %w", v, err)
		}
		l[i] = *raw
	}
	return &Prop{Raw: l}, nil
}

func (p *Prop) Get(name xml.Name) *RawXMLValue {
	for i := range p.Raw {
		raw := &p.Raw[i]
		if n, ok := raw.XMLName(); ok && name == n {
			return raw
		}
	}
	return nil
}

func (p *Prop) Decode(v any) error {
	name, err := valueXMLName(v)
	if err != nil {
		return fmt.Errorf("resolve property XML name for %T: %w", v, err)
	}

	raw := p.Get(name)
	if raw == nil {
		return HTTPErrorf(http.StatusNotFound, "missing property %s", name)
	}

	if err := raw.Decode(v); err != nil {
		return fmt.Errorf("decode property <%s %s>: %w", name.Space, name.Local, err)
	}
	return nil
}

// https://tools.ietf.org/html/rfc4918#section-14.20
type PropFind struct {
	XMLName  xml.Name  `xml:"DAV: propfind"`
	Prop     *Prop     `xml:"prop,omitzero"`
	AllProp  *struct{} `xml:"allprop,omitzero"`
	Include  *Include  `xml:"include,omitzero"`
	PropName *struct{} `xml:"propname,omitzero"`
}

func xmlNamesToRaw(names []xml.Name) []RawXMLValue {
	l := make([]RawXMLValue, len(names))
	for i, name := range names {
		l[i] = *NewRawXMLElement(name, nil, nil)
	}
	return l
}

func NewPropNamePropFind(names ...xml.Name) *PropFind {
	return &PropFind{Prop: &Prop{Raw: xmlNamesToRaw(names)}}
}

var FileInfoPropFind = NewPropNamePropFind(
	ResourceTypeName,
	GetContentLengthName,
	GetLastModifiedName,
	GetContentTypeName,
	GetETagName,
)

// https://tools.ietf.org/html/rfc4918#section-14.8
type Include struct {
	XMLName xml.Name      `xml:"DAV: include"`
	Raw     []RawXMLValue `xml:",any"`
}

// https://tools.ietf.org/html/rfc4918#section-15.9
type ResourceType struct {
	XMLName xml.Name      `xml:"DAV: resourcetype"`
	Raw     []RawXMLValue `xml:",any"`
}

func NewResourceType(names ...xml.Name) *ResourceType {
	return &ResourceType{Raw: xmlNamesToRaw(names)}
}

func (t *ResourceType) Is(name xml.Name) bool {
	for _, raw := range t.Raw {
		if n, ok := raw.XMLName(); ok && name == n {
			return true
		}
	}
	return false
}

var CollectionName = xml.Name{Space: Namespace, Local: "collection"}

// https://tools.ietf.org/html/rfc4918#section-15.4
type GetContentLength struct {
	XMLName xml.Name `xml:"DAV: getcontentlength"`
	Length  int64    `xml:",chardata"`
}

// https://tools.ietf.org/html/rfc4918#section-15.5
type GetContentType struct {
	XMLName xml.Name `xml:"DAV: getcontenttype"`
	Type    string   `xml:",chardata"`
}

type Time time.Time

func (t *Time) UnmarshalText(b []byte) error {
	tt, err := http.ParseTime(string(b))
	if err != nil {
		return fmt.Errorf("parse HTTP time %q: %w", string(b), err)
	}
	*t = Time(tt)
	return nil
}

func (t *Time) MarshalText() ([]byte, error) {
	s := time.Time(*t).UTC().Format(http.TimeFormat)
	return []byte(s), nil
}

// https://tools.ietf.org/html/rfc4918#section-15.7
type GetLastModified struct {
	XMLName      xml.Name `xml:"DAV: getlastmodified"`
	LastModified Time     `xml:",chardata"`
}

// https://tools.ietf.org/html/rfc4918#section-15.6
type GetETag struct {
	XMLName xml.Name `xml:"DAV: getetag"`
	ETag    ETag     `xml:",chardata"`
}

type ETag string

func (etag *ETag) UnmarshalText(b []byte) error {
	s, err := strconv.Unquote(string(b))
	if err != nil {
		return fmt.Errorf("webdav: failed to unquote ETag: %w", err)
	}
	*etag = ETag(s)
	return nil
}

func (etag ETag) MarshalText() ([]byte, error) {
	return []byte(etag.String()), nil
}

func (etag ETag) String() string {
	return fmt.Sprintf("%q", string(etag))
}

// https://tools.ietf.org/html/rfc4918#section-14.5
type Error struct {
	XMLName xml.Name      `xml:"DAV: error"`
	Raw     []RawXMLValue `xml:",any"`
}

func (err *Error) Error() string {
	b, _ := xml.Marshal(err)
	return string(b)
}

// https://tools.ietf.org/html/rfc4918#section-15.2
type DisplayName struct {
	XMLName xml.Name `xml:"DAV: displayname"`
	Name    string   `xml:",chardata"`
}

// https://tools.ietf.org/html/rfc5397#section-3
type CurrentUserPrincipal struct {
	XMLName         xml.Name  `xml:"DAV: current-user-principal"`
	Href            Href      `xml:"href,omitzero"`
	Unauthenticated *struct{} `xml:"unauthenticated,omitzero"`
}

// https://tools.ietf.org/html/rfc4918#section-14.19
type PropertyUpdate struct {
	XMLName xml.Name `xml:"DAV: propertyupdate"`
	Remove  []Remove `xml:"remove"`
	Set     []Set    `xml:"set"`
}

// https://tools.ietf.org/html/rfc4918#section-14.23
type Remove struct {
	XMLName xml.Name `xml:"DAV: remove"`
	Prop    Prop     `xml:"prop"`
}

// https://tools.ietf.org/html/rfc4918#section-14.26
type Set struct {
	XMLName xml.Name `xml:"DAV: set"`
	Prop    Prop     `xml:"prop"`
}

// https://tools.ietf.org/html/rfc6578#section-6.1
type SyncCollectionQuery struct {
	XMLName   xml.Name `xml:"DAV: sync-collection"`
	SyncToken string   `xml:"sync-token"`
	Limit     *Limit   `xml:"limit,omitzero"`
	SyncLevel string   `xml:"sync-level"`
	Prop      *Prop    `xml:"prop"`
}

// https://tools.ietf.org/html/rfc5323#section-5.17
type Limit struct {
	XMLName  xml.Name `xml:"DAV: limit"`
	NResults uint     `xml:"nresults"`
}

// Depth 表示请求是否作用于资源成员，定义见 RFC 4918 section 10.2。
type Depth int

const (
	// DepthZero 表示请求只作用于资源本身。
	DepthZero Depth = 0
	// DepthOne 表示请求作用于资源本身及其直接成员。
	DepthOne Depth = 1
	// DepthInfinity 表示请求作用于资源本身及所有成员。
	DepthInfinity Depth = -1
)

// ParseDepth 解析 Depth 请求头。
func ParseDepth(s string) (Depth, error) {
	switch s {
	case "0":
		return DepthZero, nil
	case "1":
		return DepthOne, nil
	case "infinity":
		return DepthInfinity, nil
	}
	return 0, fmt.Errorf("webdav: invalid Depth value")
}

// String 格式化 Depth。
func (d Depth) String() string {
	switch d {
	case DepthZero:
		return "0"
	case DepthOne:
		return "1"
	case DepthInfinity:
		return "infinity"
	}
	panic("webdav: invalid Depth value")
}

// ParseOverwrite 解析 Overwrite 请求头。
func ParseOverwrite(s string) (bool, error) {
	switch s {
	case "T":
		return true, nil
	case "F":
		return false, nil
	}
	return false, fmt.Errorf("webdav: invalid Overwrite value")
}

// FormatOverwrite 格式化 Overwrite 请求头。
func FormatOverwrite(overwrite bool) string {
	if overwrite {
		return "T"
	} else {
		return "F"
	}
}
