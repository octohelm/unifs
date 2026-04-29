package units

import (
	"bytes"
	"fmt"

	"k8s.io/apimachinery/pkg/api/resource"
)

// BinarySize 标准单位
//
// 参考: http://en.wikipedia.org/wiki/Binary_prefix
type BinarySize uint64

const (
	KB BinarySize = 1000
	MB            = 1000 * KB
	GB            = 1000 * MB
	TB            = 1000 * GB
	PB            = 1000 * TB

	KiB BinarySize = 1024
	MiB            = 1024 * KiB
	GiB            = 1024 * MiB
	TiB            = 1024 * GiB
	PiB            = 1024 * TiB
)

// IsZero 判断大小是否为零。
func (v BinarySize) IsZero() bool {
	return v == 0
}

// UnmarshalText 解析 Kubernetes 资源量字符串。
func (v *BinarySize) UnmarshalText(b []byte) error {
	if len(b) == 0 {
		return nil
	}

	size := string(bytes.TrimSuffix(b, []byte("B")))

	q, err := resource.ParseQuantity(size)
	if err != nil {
		return fmt.Errorf("parse binary size %q: %w", string(b), err)
	}

	*v = BinarySize(q.Value())

	return nil
}

// Quantity 将大小转换为 Kubernetes 资源量。
func (v BinarySize) Quantity() *resource.Quantity {
	return resource.NewQuantity(int64(v), resource.BinarySI)
}

// String 将大小格式化为 Kubernetes 二进制单位资源量。
func (v BinarySize) String() string {
	return v.Quantity().String()
}

// MarshalText 将大小格式化为文本。
func (v BinarySize) MarshalText() ([]byte, error) {
	return []byte(v.Quantity().String()), nil
}
