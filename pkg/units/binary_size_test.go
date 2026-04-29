package units_test

import (
	"fmt"
	"testing"

	. "github.com/octohelm/x/testing/v2"

	"github.com/octohelm/unifs/pkg/units"
)

func TestBinarySize(t *testing.T) {
	cases := []struct {
		request string
		expect  units.BinarySize
	}{
		{
			request: "1k",
			expect:  1 * units.KB,
		},
		{
			request: "1M",
			expect:  1 * units.MB,
		},
		{
			request: "1G",
			expect:  1 * units.GB,
		},
		{
			request: "1Gi",
			expect:  1 * units.GiB,
		},

		{
			request: "10GiB",
			expect:  10 * units.GiB,
		},
	}

	for _, c := range cases {
		t.Run(fmt.Sprintf("request %s should as %s", c.request, c.expect), func(t *testing.T) {
			var b units.BinarySize
			err := b.UnmarshalText([]byte(c.request))
			Then(t, "解析二进制大小",
				Expect(err, Equal[error](nil)),
				Expect(b, Equal(c.expect)),
			)
		})
	}
}

func TestBinarySizeText(t *testing.T) {
	size := units.BinarySize(10 * units.MiB)
	text := MustValue(t, size.MarshalText)

	Then(t, "格式化二进制大小",
		Expect(size.IsZero(), Equal(false)),
		Expect(units.BinarySize(0).IsZero(), Equal(true)),
		Expect(string(text), Equal("10Mi")),
	)

	var empty units.BinarySize
	Then(t, "空文本保持零值",
		ExpectMust(func() error {
			return empty.UnmarshalText(nil)
		}),
		Expect(empty, Equal(units.BinarySize(0))),
	)
}
