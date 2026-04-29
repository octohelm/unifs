package client

import (
	"encoding/xml"
	"fmt"
	"io"
	"reflect"
	"strings"
)

// RawXMLValue 表示原始 XML 值，实现 xml.Unmarshaler 和 xml.Marshaler，
// 可用于延迟 XML 解码或预先计算 XML 编码。
type RawXMLValue struct {
	tok      xml.Token // 保证不会是 xml.EndElement。
	children []RawXMLValue

	// encoding/xml 没有提供 TokenWriter，因此这里缓存待输出数据。
	out any
}

// NewRawXMLElement 为元素创建 RawXMLValue。
func NewRawXMLElement(name xml.Name, attr []xml.Attr, children []RawXMLValue) *RawXMLValue {
	return &RawXMLValue{tok: xml.StartElement{Name: name, Attr: attr}, children: children}
}

// EncodeRawXMLElement 将值编码到新的 RawXMLValue；该 XML 值只能用于序列化。
func EncodeRawXMLElement(v any) (*RawXMLValue, error) {
	return &RawXMLValue{out: v}, nil
}

// UnmarshalXML 实现 xml.Unmarshaler。
func (val *RawXMLValue) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	val.tok = start
	val.children = nil
	val.out = nil

	for {
		tok, err := d.Token()
		if err != nil {
			return fmt.Errorf("read XML token for <%s %s>: %w", start.Name.Space, start.Name.Local, err)
		}
		switch tok := tok.(type) {
		case xml.StartElement:
			child := RawXMLValue{}
			if err := child.UnmarshalXML(d, tok); err != nil {
				return fmt.Errorf("decode child XML element <%s %s>: %w", tok.Name.Space, tok.Name.Local, err)
			}
			val.children = append(val.children, child)
		case xml.EndElement:
			return nil
		default:
			val.children = append(val.children, RawXMLValue{tok: xml.CopyToken(tok)})
		}
	}
}

// MarshalXML 实现 xml.Marshaler。
func (val *RawXMLValue) MarshalXML(e *xml.Encoder, start xml.StartElement) error {
	if val.out != nil {
		if err := e.Encode(val.out); err != nil {
			return fmt.Errorf("encode XML value %T: %w", val.out, err)
		}
		return nil
	}

	switch tok := val.tok.(type) {
	case xml.StartElement:
		if err := e.EncodeToken(tok); err != nil {
			return fmt.Errorf("encode XML start <%s %s>: %w", tok.Name.Space, tok.Name.Local, err)
		}
		for _, child := range val.children {
			// TODO 为起始参数选择更合适的值。
			if err := child.MarshalXML(e, xml.StartElement{}); err != nil {
				return fmt.Errorf("encode XML child of <%s %s>: %w", tok.Name.Space, tok.Name.Local, err)
			}
		}
		if err := e.EncodeToken(tok.End()); err != nil {
			return fmt.Errorf("encode XML end <%s %s>: %w", tok.Name.Space, tok.Name.Local, err)
		}
		return nil
	case xml.EndElement:
		panic("unexpected end element")
	default:
		if err := e.EncodeToken(tok); err != nil {
			return fmt.Errorf("encode XML token %T: %w", tok, err)
		}
		return nil
	}
}

var (
	_ xml.Marshaler   = (*RawXMLValue)(nil)
	_ xml.Unmarshaler = (*RawXMLValue)(nil)
)

func (val *RawXMLValue) Decode(v any) error {
	if err := xml.NewTokenDecoder(val.TokenReader()).Decode(v); err != nil {
		return fmt.Errorf("decode raw XML into %T: %w", v, err)
	}
	return nil
}

func (val *RawXMLValue) XMLName() (name xml.Name, ok bool) {
	if start, ok := val.tok.(xml.StartElement); ok {
		return start.Name, true
	}
	return xml.Name{}, false
}

// TokenReader 返回该 XML 值的 token 流。
func (val *RawXMLValue) TokenReader() xml.TokenReader {
	if val.out != nil {
		panic("webdav: called RawXMLValue.TokenReader on a marshal-only XML value")
	}
	return &rawXMLValueReader{val: val}
}

type rawXMLValueReader struct {
	val         *RawXMLValue
	start, end  bool
	child       int
	childReader xml.TokenReader
}

func (tr *rawXMLValueReader) Token() (xml.Token, error) {
	if tr.end {
		return nil, io.EOF
	}

	start, ok := tr.val.tok.(xml.StartElement)
	if !ok {
		tr.end = true
		return tr.val.tok, nil
	}

	if !tr.start {
		tr.start = true
		return start, nil
	}

	for tr.child < len(tr.val.children) {
		if tr.childReader == nil {
			tr.childReader = tr.val.children[tr.child].TokenReader()
		}

		tok, err := tr.childReader.Token()
		if err == io.EOF {
			tr.childReader = nil
			tr.child++
		} else {
			if err != nil {
				return nil, fmt.Errorf("read child XML token: %w", err)
			}
			return tok, nil
		}
	}

	tr.end = true
	return start.End(), nil
}

var _ xml.TokenReader = (*rawXMLValueReader)(nil)

func valueXMLName(v any) (xml.Name, error) {
	t := reflect.TypeOf(v)
	for t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	if t.Kind() != reflect.Struct {
		return xml.Name{}, fmt.Errorf("webdav: %T is not a struct", v)
	}
	nameField, ok := t.FieldByName("XMLName")
	if !ok {
		return xml.Name{}, fmt.Errorf("webdav: %T is missing an XMLName struct field", v)
	}
	if nameField.Type != reflect.TypeFor[xml.Name]() {
		return xml.Name{}, fmt.Errorf("webdav: %T.XMLName isn't an xml.Name", v)
	}
	tag := nameField.Tag.Get("xml")
	if tag == "" {
		return xml.Name{}, fmt.Errorf(`webdav: %T.XMLName is missing an "xml" tag`, v)
	}
	name := strings.Split(tag, ",")[0]
	nameParts := strings.Split(name, " ")
	if len(nameParts) != 2 {
		return xml.Name{}, fmt.Errorf("webdav: expected a namespace and local name in %T.XMLName's xml tag", v)
	}
	return xml.Name{Space: nameParts[0], Local: nameParts[1]}, nil
}
