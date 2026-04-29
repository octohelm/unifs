package strfmt

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"
)

// ParseEndpoint 将文本解析为 Endpoint。
func ParseEndpoint(text string) (*Endpoint, error) {
	u, err := url.ParseRequestURI(text)
	if err != nil {
		return nil, fmt.Errorf("parse endpoint %q: %w", text, err)
	}

	endpoint := &Endpoint{
		Scheme: u.Scheme,
	}

	query := u.Query()

	if len(query) > 0 {
		endpoint.Extra = u.Query()
	}

	endpoint.Path = u.Path

	endpoint.Hostname = u.Hostname()

	i, err := strconv.ParseUint(u.Port(), 10, 16)
	if err == nil {
		endpoint.Port = uint16(i)
	}

	if u.User != nil {
		endpoint.Username = u.User.Username()
		endpoint.Password, _ = u.User.Password()
	}

	return endpoint, nil
}

// Endpoint 描述类似 URL 的后端端点。
//
// openapi:strfmt endpoint
type Endpoint struct {
	Scheme   string
	Hostname string
	Port     uint16
	Path     string
	Username string
	Password string
	Extra    url.Values
}

// Base 返回不带前导斜杠的第一个路径片段。
func (e Endpoint) Base() string {
	if e.Path != "" {
		return strings.Split(e.Path[1:], "/")[0]
	}
	return ""
}

// IsZero 判断端点是否未设置协议名。
func (e Endpoint) IsZero() bool {
	return e.Scheme == ""
}

// SecurityString 格式化端点，并隐藏密码。
func (e Endpoint) SecurityString() string {
	e.Password = strings.Repeat("-", len(e.Password))
	return e.String()
}

// Host 返回主机名，存在端口时返回 host:port。
func (e Endpoint) Host() string {
	if e.Port != 0 {
		return e.Hostname + ":" + strconv.FormatUint(uint64(e.Port), 10)
	}
	return e.Hostname
}

// String 将端点格式化为 URL 字符串。
func (e Endpoint) String() string {
	u := url.URL{}
	u.Scheme = e.Scheme
	u.Host = e.Host()

	if e.Extra != nil {
		u.RawQuery = e.Extra.Encode()
	}

	u.Path = e.Path

	if e.Username != "" || e.Password != "" {
		u.User = url.UserPassword(e.Username, e.Password)
	}

	return u.String()
}

// IsTLS 判断端点协议名是否以 "s" 结尾。
func (e *Endpoint) IsTLS() bool {
	if e.Scheme == "" {
		return false
	}
	return e.Scheme[len(e.Scheme)-1] == 's'
}

// UnmarshalText 解析文本端点。
func (e *Endpoint) UnmarshalText(text []byte) error {
	endpoint, err := ParseEndpoint(string(text))
	if err != nil {
		return fmt.Errorf("unmarshal endpoint: %w", err)
	}
	*e = *endpoint
	return nil
}

// MarshalText 将端点格式化为文本。
func (e Endpoint) MarshalText() (text []byte, err error) {
	return []byte(e.String()), nil
}
