package units

import (
	"fmt"
	"testing"

	testingx "github.com/octohelm/x/testing"
)

func TestBinarySize(t *testing.T) {
	cases := []struct {
		request string
		expect  BinarySize
	}{
		{
			request: "1k",
			expect:  1 * KB,
		},
		{
			request: "1M",
			expect:  1 * MB,
		},
		{
			request: "1G",
			expect:  1 * GB,
		},
		{
			request: "1Gi",
			expect:  1 * GiB,
		},

		{
			request: "10GiB",
			expect:  10 * GiB,
		},
	}

	for _, c := range cases {
		t.Run(fmt.Sprintf("request %s should as %s", c.request, c.expect), func(t *testing.T) {
			var b BinarySize
			err := b.UnmarshalText([]byte(c.request))
			testingx.Expect(t, err, testingx.BeNil[error]())
			testingx.Expect(t, b, testingx.Be(c.expect))
		})
	}
}
