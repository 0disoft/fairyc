package sourcenotice

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func noticeFile(t *testing.T, root, name, text string) {
	t.Helper()
	path := filepath.Join(root, name)
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(text), 0600); err != nil {
		t.Fatal(err)
	}
}

func TestNoticeThresholdIsStrictBytesAndDoesNotReadOrChangeSource(t *testing.T) {
	root := t.TempDir()
	noticeFile(t, root, "exact.fy", strings.Repeat("a", 40000))
	noticeFile(t, root, "large.fy", strings.Repeat("a", 40001))
	noticeFile(t, root, "nested/unicode.go", strings.Repeat("한", 13334))
	noticeFile(t, root, "small.FY", "invalid Fairy syntax is not parsed")
	noticeFile(t, root, "not-source.bin", strings.Repeat("x", 50000))
	noticeFile(t, root, "node_modules/ignored.ts", strings.Repeat("x", 50000))
	noticeFile(t, root, ".cache/generated.fy", strings.Repeat("x", 50000))
	path := filepath.Join(root, "large.fy")
	before, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	r := Scan(context.Background(), root, DefaultThresholdBytes)
	if r.State != "bounded" || r.SourceFiles != 4 || len(r.Files) != 2 || r.Files[0].Path != "large.fy" || r.Files[0].Bytes != 40001 || r.Files[1].Bytes != 40002 {
		t.Fatalf("notice: %+v", r)
	}
	a, _ := json.Marshal(r)
	b, _ := json.Marshal(Scan(context.Background(), root, DefaultThresholdBytes))
	if string(a) != string(b) {
		t.Fatal("nondeterministic notice order")
	}
	after, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	contents, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !before.ModTime().Equal(after.ModTime()) || string(contents) != strings.Repeat("a", 40001) {
		t.Fatal("notice changed/truncated source")
	}
}

func TestNoticeReportsIncompleteScanInsteadOfCleanResult(t *testing.T) {
	root := t.TempDir()
	noticeFile(t, root, "a.fy", strings.Repeat("x", 20))
	noticeFile(t, root, "b.fy", strings.Repeat("x", 20))
	r := scan(context.Background(), root, 10, 2)
	if r.State != "partial" || len(r.Issues) != 1 || r.Issues[0].Reason != "scan-limit" || len(r.Files) != 1 {
		t.Fatalf("bounded scan lost partial status: %+v", r)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	for _, r := range []Report{Scan(ctx, root, 10), Scan(context.Background(), root, 0), Scan(context.Background(), filepath.Join(root, "missing"), 10), Scan(context.Background(), filepath.Join(root, "a.fy"), 10)} {
		if r.State != "unknown" || len(r.Issues) == 0 {
			t.Fatalf("invalid scan passed: %+v", r)
		}
	}
}

func TestNoticeSharesBoundedBudgetAcrossManyDirectories(t *testing.T) {
	root := t.TempDir()
	noticeFile(t, root, "00-root.fy", strings.Repeat("x", 20))
	for _, name := range []string{"a/one.fy", "a/two.fy", "b/one.fy", "b/two.fy", "c/one.fy"} {
		noticeFile(t, root, name, strings.Repeat("x", 20))
	}
	r := scan(context.Background(), root, 10, 8)
	if r.State != "partial" || r.VisitedEntries != 8 || r.SourceFiles != 4 || len(r.Files) != 4 || len(r.Issues) != 1 || r.Issues[0].Reason != "scan-limit" {
		t.Fatalf("multi-directory bounded scan = %+v", r)
	}
	for index := 1; index < len(r.Files); index++ {
		if r.Files[index-1].Path >= r.Files[index].Path {
			t.Fatalf("partial findings are not sorted: %+v", r.Files)
		}
	}
}

func TestNoticeSkipsSymlinkWithoutFollowingIt(t *testing.T) {
	root, outside := t.TempDir(), t.TempDir()
	noticeFile(t, outside, "outside.fy", strings.Repeat("x", 50000))
	if err := os.Symlink(outside, filepath.Join(root, "external")); err != nil {
		t.Skipf("host cannot create test symlink: %v", err)
	}
	r := Scan(context.Background(), root, 40000)
	if r.SourceFiles != 0 || len(r.Files) != 0 || len(r.SkippedSymlinks) != 1 || r.SkippedSymlinks[0] != "external" {
		t.Fatalf("symlink followed: %+v", r)
	}
}
