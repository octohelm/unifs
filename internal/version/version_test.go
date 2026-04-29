package version

import (
	"testing"

	. "github.com/octohelm/x/testing/v2"
)

func TestVersion(t *testing.T) {
	Then(t, "返回默认构建版本",
		Expect(Version(), Equal("v0.0.0")),
	)
}
