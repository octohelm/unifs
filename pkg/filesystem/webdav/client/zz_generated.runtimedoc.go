/*
Package client GENERATED BY gengo:runtimedoc 
DON'T EDIT THIS FILE
*/
package client

import _ "embed"

// nolint:deadcode,unused
func runtimeDoc(v any, prefix string, names ...string) ([]string, bool) {
	if c, ok := v.(interface {
		RuntimeDoc(names ...string) ([]string, bool)
	}); ok {
		doc, ok := c.RuntimeDoc(names...)
		if ok {
			if prefix != "" && len(doc) > 0 {
				doc[0] = prefix + doc[0]
				return doc, true
			}

			return doc, true
		}
	}
	return nil, false
}

func (v *CurrentUserPrincipal) RuntimeDoc(names ...string) ([]string, bool) {
	if len(names) > 0 {
		switch names[0] {
		case "XMLName":
			return []string{}, true
		case "Href":
			return []string{}, true
		case "Unauthenticated":
			return []string{}, true

		}

		return nil, false
	}
	return []string{
		"https://tools.ietf.org/html/rfc5397#section-3",
	}, true
}

func (*Depth) RuntimeDoc(names ...string) ([]string, bool) {
	return []string{
		"indicates whether a request applies to the resource's members. It's",
		"defined in RFC 4918 section 10.2.",
	}, true
}

func (v *DisplayName) RuntimeDoc(names ...string) ([]string, bool) {
	if len(names) > 0 {
		switch names[0] {
		case "XMLName":
			return []string{}, true
		case "Name":
			return []string{}, true

		}

		return nil, false
	}
	return []string{
		"https://tools.ietf.org/html/rfc4918#section-15.2",
	}, true
}

func (*ETag) RuntimeDoc(names ...string) ([]string, bool) {
	return []string{}, true
}

func (v *Error) RuntimeDoc(names ...string) ([]string, bool) {
	if len(names) > 0 {
		switch names[0] {
		case "XMLName":
			return []string{}, true
		case "Raw":
			return []string{}, true

		}

		return nil, false
	}
	return []string{
		"https://tools.ietf.org/html/rfc4918#section-14.5",
	}, true
}

func (v *GetContentLength) RuntimeDoc(names ...string) ([]string, bool) {
	if len(names) > 0 {
		switch names[0] {
		case "XMLName":
			return []string{}, true
		case "Length":
			return []string{}, true

		}

		return nil, false
	}
	return []string{
		"https://tools.ietf.org/html/rfc4918#section-15.4",
	}, true
}

func (v *GetContentType) RuntimeDoc(names ...string) ([]string, bool) {
	if len(names) > 0 {
		switch names[0] {
		case "XMLName":
			return []string{}, true
		case "Type":
			return []string{}, true

		}

		return nil, false
	}
	return []string{
		"https://tools.ietf.org/html/rfc4918#section-15.5",
	}, true
}

func (v *GetETag) RuntimeDoc(names ...string) ([]string, bool) {
	if len(names) > 0 {
		switch names[0] {
		case "XMLName":
			return []string{}, true
		case "ETag":
			return []string{}, true

		}

		return nil, false
	}
	return []string{
		"https://tools.ietf.org/html/rfc4918#section-15.6",
	}, true
}

func (v *GetLastModified) RuntimeDoc(names ...string) ([]string, bool) {
	if len(names) > 0 {
		switch names[0] {
		case "XMLName":
			return []string{}, true
		case "LastModified":
			return []string{}, true

		}

		return nil, false
	}
	return []string{
		"https://tools.ietf.org/html/rfc4918#section-15.7",
	}, true
}

func (v *HTTPError) RuntimeDoc(names ...string) ([]string, bool) {
	if len(names) > 0 {
		switch names[0] {
		case "Code":
			return []string{}, true
		case "Err":
			return []string{}, true

		}

		return nil, false
	}
	return []string{}, true
}

func (v *Href) RuntimeDoc(names ...string) ([]string, bool) {
	if len(names) > 0 {
		switch names[0] {
		case "Scheme":
			return []string{}, true
		case "Opaque":
			return []string{}, true
		case "User":
			return []string{
				"encoded opaque data",
			}, true
		case "Host":
			return []string{
				"username and password information",
			}, true
		case "Path":
			return []string{
				"host or host:port (see Hostname and Port methods)",
			}, true
		case "RawPath":
			return []string{
				"path (relative paths may omit leading slash)",
			}, true
		case "OmitHost":
			return []string{
				"encoded path hint (see EscapedPath method)",
			}, true
		case "ForceQuery":
			return []string{
				"do not emit empty host (authority)",
			}, true
		case "RawQuery":
			return []string{
				"append a query ('?') even if RawQuery is empty",
			}, true
		case "Fragment":
			return []string{
				"encoded query values, without '?'",
			}, true
		case "RawFragment":
			return []string{
				"fragment for references, without '#'",
			}, true

		}

		return nil, false
	}
	return []string{}, true
}

func (v *Include) RuntimeDoc(names ...string) ([]string, bool) {
	if len(names) > 0 {
		switch names[0] {
		case "XMLName":
			return []string{}, true
		case "Raw":
			return []string{}, true

		}

		return nil, false
	}
	return []string{
		"https://tools.ietf.org/html/rfc4918#section-14.8",
	}, true
}

func (v *Limit) RuntimeDoc(names ...string) ([]string, bool) {
	if len(names) > 0 {
		switch names[0] {
		case "XMLName":
			return []string{}, true
		case "NResults":
			return []string{}, true

		}

		return nil, false
	}
	return []string{
		"https://tools.ietf.org/html/rfc5323#section-5.17",
	}, true
}

func (v *Location) RuntimeDoc(names ...string) ([]string, bool) {
	if len(names) > 0 {
		switch names[0] {
		case "XMLName":
			return []string{}, true
		case "Href":
			return []string{}, true

		}

		return nil, false
	}
	return []string{
		"https://tools.ietf.org/html/rfc4918#section-14.9",
	}, true
}

func (v *MultiStatus) RuntimeDoc(names ...string) ([]string, bool) {
	if len(names) > 0 {
		switch names[0] {
		case "XMLName":
			return []string{}, true
		case "Responses":
			return []string{}, true
		case "ResponseDescription":
			return []string{}, true
		case "SyncToken":
			return []string{}, true

		}

		return nil, false
	}
	return []string{
		"https://tools.ietf.org/html/rfc4918#section-14.16",
	}, true
}

func (v *Prop) RuntimeDoc(names ...string) ([]string, bool) {
	if len(names) > 0 {
		switch names[0] {
		case "XMLName":
			return []string{}, true
		case "Raw":
			return []string{}, true

		}

		return nil, false
	}
	return []string{
		"https://tools.ietf.org/html/rfc4918#section-14.18",
	}, true
}

func (v *PropFind) RuntimeDoc(names ...string) ([]string, bool) {
	if len(names) > 0 {
		switch names[0] {
		case "XMLName":
			return []string{}, true
		case "Prop":
			return []string{}, true
		case "AllProp":
			return []string{}, true
		case "Include":
			return []string{}, true
		case "PropName":
			return []string{}, true

		}

		return nil, false
	}
	return []string{
		"https://tools.ietf.org/html/rfc4918#section-14.20",
	}, true
}

func (v *PropStat) RuntimeDoc(names ...string) ([]string, bool) {
	if len(names) > 0 {
		switch names[0] {
		case "XMLName":
			return []string{}, true
		case "Prop":
			return []string{}, true
		case "Status":
			return []string{}, true
		case "ResponseDescription":
			return []string{}, true
		case "Error":
			return []string{}, true

		}

		return nil, false
	}
	return []string{
		"https://tools.ietf.org/html/rfc4918#section-14.22",
	}, true
}

func (v *PropertyUpdate) RuntimeDoc(names ...string) ([]string, bool) {
	if len(names) > 0 {
		switch names[0] {
		case "XMLName":
			return []string{}, true
		case "Remove":
			return []string{}, true
		case "Set":
			return []string{}, true

		}

		return nil, false
	}
	return []string{
		"https://tools.ietf.org/html/rfc4918#section-14.19",
	}, true
}

func (v *Remove) RuntimeDoc(names ...string) ([]string, bool) {
	if len(names) > 0 {
		switch names[0] {
		case "XMLName":
			return []string{}, true
		case "Prop":
			return []string{}, true

		}

		return nil, false
	}
	return []string{
		"https://tools.ietf.org/html/rfc4918#section-14.23",
	}, true
}

func (v *ResourceType) RuntimeDoc(names ...string) ([]string, bool) {
	if len(names) > 0 {
		switch names[0] {
		case "XMLName":
			return []string{}, true
		case "Raw":
			return []string{}, true

		}

		return nil, false
	}
	return []string{
		"https://tools.ietf.org/html/rfc4918#section-15.9",
	}, true
}

func (v *Response) RuntimeDoc(names ...string) ([]string, bool) {
	if len(names) > 0 {
		switch names[0] {
		case "XMLName":
			return []string{}, true
		case "Hrefs":
			return []string{}, true
		case "PropStats":
			return []string{}, true
		case "ResponseDescription":
			return []string{}, true
		case "Status":
			return []string{}, true
		case "Error":
			return []string{}, true
		case "Location":
			return []string{}, true
		case "Prefix":
			return []string{}, true

		}

		return nil, false
	}
	return []string{
		"https://tools.ietf.org/html/rfc4918#section-14.24",
	}, true
}

func (v *Set) RuntimeDoc(names ...string) ([]string, bool) {
	if len(names) > 0 {
		switch names[0] {
		case "XMLName":
			return []string{}, true
		case "Prop":
			return []string{}, true

		}

		return nil, false
	}
	return []string{
		"https://tools.ietf.org/html/rfc4918#section-14.26",
	}, true
}

func (v *Status) RuntimeDoc(names ...string) ([]string, bool) {
	if len(names) > 0 {
		switch names[0] {
		case "Code":
			return []string{}, true
		case "Text":
			return []string{}, true

		}

		return nil, false
	}
	return []string{}, true
}

func (v *SyncCollectionQuery) RuntimeDoc(names ...string) ([]string, bool) {
	if len(names) > 0 {
		switch names[0] {
		case "XMLName":
			return []string{}, true
		case "SyncToken":
			return []string{}, true
		case "Limit":
			return []string{}, true
		case "SyncLevel":
			return []string{}, true
		case "Prop":
			return []string{}, true

		}

		return nil, false
	}
	return []string{
		"https://tools.ietf.org/html/rfc6578#section-6.1",
	}, true
}
