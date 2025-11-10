package client

import (
	"net/url"
)

type Href url.URL

func (h *Href) String() string {
	u := (*url.URL)(h)
	return u.String()
}

func (h *Href) MarshalText() ([]byte, error) {
	return []byte(h.String()), nil
}

func (h *Href) UnmarshalText(b []byte) error {
	u, err := url.Parse(string(b))
	if err != nil {
		return err
	}
	*h = Href(*u)
	return nil
}
