package units

import (
	"strings"

	"k8s.io/apimachinery/pkg/api/resource"
)

// BinarySize 标准单位
//
// See: http://en.wikipedia.org/wiki/Binary_prefix
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

func (v BinarySize) IsZero() bool {
	return v == 0
}

func (v *BinarySize) UnmarshalText(b []byte) error {
	if len(b) == 0 {
		return nil
	}

	size := string(b)
	if strings.HasSuffix(size, "B") {
		size = size[0 : len(size)-1]
	}

	q, err := resource.ParseQuantity(size)
	if err != nil {
		return err
	}

	*v = BinarySize(q.Value())

	return nil
}

func (v BinarySize) Quantity() *resource.Quantity {
	return resource.NewQuantity(int64(v), resource.BinarySI)
}

func (v BinarySize) String() string {
	return v.Quantity().String()
}

func (v BinarySize) MarshalText() ([]byte, error) {
	return []byte(v.Quantity().String()), nil
}
