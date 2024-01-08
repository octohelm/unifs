package testutil

import (
	"context"
	"os"
	"path"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"testing"

	"github.com/octohelm/unifs/pkg/filesystem"

	"golang.org/x/net/webdav"

	_ "unsafe"
)

const (
	infiniteDepth = -1
)

func TestFullFS(t *testing.T, fs filesystem.FileSystem) {
	errStr := func(err error) string {
		switch {
		case os.IsExist(err):
			return "errExist"
		case os.IsNotExist(err):
			return "errNotExist"
		case err != nil:
			return "err"
		}
		return "ok"
	}

	// The non-"find" non-"stat" test cases should change the file system state. The
	// indentation of the "find"s and "stat"s helps distinguish such test cases.
	testCases := []string{
		"  stat / want dir",
		"  stat /a want errNotExist",
		"  stat /d want errNotExist",
		"  stat /d/e want errNotExist",
		"create /a A want ok",
		"  stat /a want 1",
		"create /d/e EEE want errNotExist",
		"mk-dir /a want errExist",
		"mk-dir /d/m want errNotExist",
		"mk-dir /d want ok",
		"  stat /d want dir",
		"create /d/e EEE want ok",
		"  stat /d/e want 3",
		"  find / /a /d /d/e",
		"create /d/f FFFF want ok",
		"create /d/g GGGGGGG want ok",
		"mk-dir /d/m want ok",
		"mk-dir /d/m want errExist",
		"create /d/m/p PPPPP want ok",
		"  stat /d/e want 3",
		"  stat /d/f want 4",
		"  stat /d/g want 7",
		"  stat /d/h want errNotExist",
		"  stat /d/m want dir",
		"  stat /d/m/p want 5",
		"  find / /a /d /d/e /d/f /d/g /d/m /d/m/p",
		"rm-all /d want ok",
		"  stat /a want 1",
		"  stat /d want errNotExist",
		"  stat /d/e want errNotExist",
		"  stat /d/f want errNotExist",
		"  stat /d/g want errNotExist",
		"  stat /d/m want errNotExist",
		"  stat /d/m/p want errNotExist",
		"  find / /a",
		"mk-dir /d/m want errNotExist",
		"mk-dir /d want ok",
		"create /d/f FFFF want ok",
		"rm-all /d/f want ok",
		"mk-dir /d/m want ok",
		"rm-all /z want ok",
		"rm-all / want err",
		"create /b BB want ok",
		"  stat / want dir",
		"  stat /a want 1",
		"  stat /b want 2",
		"  stat /c want errNotExist",
		"  stat /d want dir",
		"  stat /d/m want dir",
		"  find / /a /b /d /d/m",
		"move__ o=F /b /c want ok",
		"  stat /b want errNotExist",
		"  stat /c want 2",
		"  stat /d/m want dir",
		"  stat /d/n want errNotExist",
		"  find / /a /c /d /d/m",
		"move__ o=F /d/m /d/n want ok",
		"create /d/n/q QQQQ want ok",
		"  stat /d/m want errNotExist",
		"  stat /d/n want dir",
		"  stat /d/n/q want 4",
		"move__ o=F /d /d/n/z want err",
		"move__ o=T /c /d/n/q want ok",
		"  stat /c want errNotExist",
		"  stat /d/n/q want 2",
		"  find / /a /d /d/n /d/n/q",
		"create /d/n/r RRRRR want ok",
		"mk-dir /u want ok",
		"mk-dir /u/v want ok",
		"move__ o=F /d/n /u want errExist",
		"create /t TTTTTT want ok",
		"move__ o=F /d/n /t want errExist",
		"rm-all /t want ok",
		"move__ o=F /d/n /t want ok",
		"  stat /d want dir",
		"  stat /d/n want errNotExist",
		"  stat /d/n/r want errNotExist",
		"  stat /t want dir",
		"  stat /t/q want 2",
		"  stat /t/r want 5",
		"  find / /a /d /t /t/q /t/r /u /u/v",
		"move__ o=F /t / want errExist",
		"move__ o=T /t /u/v want ok",
		"  stat /u/v/r want 5",
		"move__ o=F / /z want err",
		"  find / /a /d /u /u/v /u/v/q /u/v/r",
		"  stat /a want 1",
		"  stat /b want errNotExist",
		"  stat /c want errNotExist",
		"  stat /u/v/r want 5",
		"copy__ o=F d=0 /a /b want ok",
		"copy__ o=T d=0 /a /c want ok",
		"  stat /a want 1",
		"  stat /b want 1",
		"  stat /c want 1",
		"  stat /u/v/r want 5",
		"copy__ o=F d=0 /u/v/r /b want errExist",
		"  stat /b want 1",
		"copy__ o=T d=0 /u/v/r /b want ok",
		"  stat /a want 1",
		"  stat /b want 5",
		"  stat /u/v/r want 5",
		"rm-all /a want ok",
		"rm-all /b want ok",
		"mk-dir /u/v/w want ok",
		"create /u/v/w/s SSSSSSSS want ok",
		"  stat /d want dir",
		"  stat /d/x want errNotExist",
		"  stat /d/y want errNotExist",
		"  stat /u/v/r want 5",
		"  stat /u/v/w/s want 8",
		"  find / /c /d /u /u/v /u/v/q /u/v/r /u/v/w /u/v/w/s",
		"copy__ o=T d=0 /u/v /d/x want ok",
		"copy__ o=T d=∞ /u/v /d/y want ok",
		"rm-all /u want ok",
		"  stat /d/x want dir",
		"  stat /d/x/q want errNotExist",
		"  stat /d/x/r want errNotExist",
		"  stat /d/x/w want errNotExist",
		"  stat /d/x/w/s want errNotExist",
		"  stat /d/y want dir",
		"  stat /d/y/q want 2",
		"  stat /d/y/r want 5",
		"  stat /d/y/w want dir",
		"  stat /d/y/w/s want 8",
		"  stat /u want errNotExist",
		"  find / /c /d /d/x /d/y /d/y/q /d/y/r /d/y/w /d/y/w/s",
		"copy__ o=F d=∞ /d/y /d/x want errExist",
	}

	for i := range testCases {
		tc := testCases[i]

		t.Log(tc)

		ctx := context.Background()

		tc = strings.TrimSpace(tc)
		j := strings.IndexByte(tc, ' ')
		if j < 0 {
			t.Fatalf("test case #%d %q: invalid command", i, tc)
		}
		op, arg := tc[:j], tc[j+1:]

		switch op {
		default:
			t.Fatalf("test case #%d %q: invalid operation %q", i, tc, op)

		case "create":
			parts := strings.Split(arg, " ")
			if len(parts) != 4 || parts[2] != "want" {
				t.Fatalf("test case #%d %q: invalid write", i, tc)
			}
			f, opErr := fs.OpenFile(ctx, parts[0], os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0666)
			if got := errStr(opErr); got != parts[3] {
				t.Fatalf("test case #%d %q: OpenFile: got %q (%v), want %q", i, tc, got, opErr, parts[3])
			}
			if f != nil {
				if _, err := f.Write([]byte(parts[1])); err != nil {
					t.Fatalf("test case #%d %q: Write: %v", i, tc, err)
				}
				if err := f.Close(); err != nil {
					t.Fatalf("test case #%d %q: Close: %v", i, tc, err)
				}
			}
		case "find":
			got, err := find(ctx, nil, fs, "/")
			if err != nil {
				t.Fatalf("test case #%d %q: find: %v", i, tc, err)
			}
			sort.Strings(got)
			want := strings.Split(arg, " ")
			if !reflect.DeepEqual(got, want) {
				t.Fatalf("test case #%d %q:\ngot  %s\nwant %s", i, tc, got, want)
			}

		case "copy__", "mk-dir", "move__", "rm-all", "stat":
			nParts := 3
			switch op {
			case "copy__":
				nParts = 6
			case "move__":
				nParts = 5
			}
			parts := strings.Split(arg, " ")
			if len(parts) != nParts {
				t.Fatalf("test case #%d %q: invalid %s", i, tc, op)
			}
			got, opErr := "", error(nil)
			switch op {
			case "copy__":
				depth := 0
				if parts[1] == "d=∞" {
					depth = infiniteDepth
				}
				_, opErr = copyFiles(ctx, fs, parts[2], parts[3], parts[0] == "o=T", depth, 0)
			case "mk-dir":
				opErr = fs.Mkdir(ctx, parts[0], 0777)
			case "move__":
				_, opErr = moveFiles(ctx, fs, parts[1], parts[2], parts[0] == "o=T")
			case "rm-all":
				opErr = fs.RemoveAll(ctx, parts[0])
			case "stat":
				var stat os.FileInfo
				fileName := parts[0]
				if stat, opErr = fs.Stat(ctx, fileName); opErr == nil {
					if stat.IsDir() {
						got = "dir"
					} else {
						got = strconv.Itoa(int(stat.Size()))
					}

					if fileName == "/" {
						// For a Dir FileSystem, the virtual file system root maps to a
						// real file system name like "/tmp/webdav-test012345", which does
						// not end with "/". We skip such cases.
					} else if statName := stat.Name(); path.Base(fileName) != statName {
						t.Fatalf("test case #%d %q: file name %q inconsistent with stat name %q",
							i, tc, fileName, statName)
					}
				}

			}
			if got == "" {
				got = errStr(opErr)
			}

			if parts[len(parts)-2] != "want" {
				t.Fatalf("test case #%d %q: invalid %s", i, tc, op)
			}
			if want := parts[len(parts)-1]; got != want {
				t.Fatalf("test case #%d %q: got %q (%v), want %q", i, tc, got, opErr, want)
			}
		}
	}
}

func find(ctx context.Context, ss []string, fs filesystem.FileSystem, name string) ([]string, error) {
	stat, err := fs.Stat(ctx, name)
	if err != nil {
		return nil, err
	}
	ss = append(ss, name)
	if stat.IsDir() {
		f, err := fs.OpenFile(ctx, name, os.O_RDONLY, 0)
		if err != nil {
			return nil, err
		}
		defer f.Close()
		children, err := f.Readdir(-1)
		if err != nil {
			return nil, err
		}
		for _, c := range children {
			ss, err = find(ctx, ss, fs, path.Join(name, c.Name()))
			if err != nil {
				return nil, err
			}
		}
	}
	return ss, nil
}

//go:linkname moveFiles golang.org/x/net/webdav.moveFiles
func moveFiles(ctx context.Context, fs webdav.FileSystem, src, dst string, overwrite bool) (status int, err error)

//go:linkname copyFiles golang.org/x/net/webdav.copyFiles
func copyFiles(ctx context.Context, fs webdav.FileSystem, src, dst string, overwrite bool, depth int, recursion int) (status int, err error)
