package client

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
)

type Status struct {
	Code int
	Text string
}

func (s *Status) MarshalText() ([]byte, error) {
	text := s.Text
	if text == "" {
		text = http.StatusText(s.Code)
	}
	return fmt.Appendf(nil, "HTTP/1.1 %v %v", s.Code, text), nil
}

func (s *Status) UnmarshalText(b []byte) error {
	if len(b) == 0 {
		return nil
	}

	parts := strings.SplitN(string(b), " ", 3)
	if len(parts) != 3 {
		return fmt.Errorf("webdav: invalid HTTP status %d: expected 3 fields", s.Code)
	}
	code, err := strconv.Atoi(parts[1])
	if err != nil {
		return fmt.Errorf("webdav: invalid HTTP status %d: failed to parse code: %w", s.Code, err)
	}

	s.Code = code
	s.Text = parts[2]
	return nil
}

func (s *Status) Err() error {
	if s == nil {
		return nil
	}

	// TODO 处理 2xx 和 3xx 状态码。
	if s.Code != http.StatusOK {
		return &HTTPError{Code: s.Code}
	}
	return nil
}
