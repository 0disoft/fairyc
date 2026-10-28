package lockfile

import (
	"bytes"
	"fmt"
	"path/filepath"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/workspaceloader"
)

func TestParseLockTrailingDataAndWhitespace(t *testing.T) {
	workspace, err := workspaceloader.Load(filepath.Join("..", "..", "spec", "fixtures", "workspace-loader", "cases", "pass-feature-distinct"))
	if err != nil {
		t.Fatal(err)
	}
	document, err := FromWorkspace(workspace)
	if err != nil {
		t.Fatal(err)
	}
	canonical, err := Marshal(document)
	if err != nil {
		t.Fatal(err)
	}
	// Newlines are part of the generated format; all four JSON whitespace bytes
	// must remain accepted, including when the decoder has already buffered them.
	for _, whitespace := range []string{"", " ", "\t", "\r", "\n", " \t\r\n"} {
		contents := append(bytes.Clone(canonical), whitespace...)
		parsed, err := Parse(contents)
		if err != nil {
			t.Fatalf("whitespace %q: %v", whitespace, err)
		}
		if err := MatchesWorkspace(parsed, workspace); err != nil {
			t.Fatal(err)
		}
	}
	for _, tail := range []string{"null", "false", "0", `"string"`, "{}", "[]", "[", "garbage", "\x00", "\v", "\f", "\u00a0"} {
		for _, whitespace := range []string{"", " \t\r\n"} {
			contents := append(bytes.Clone(canonical), whitespace+tail...)
			if _, err := Parse(contents); err == nil || !strings.Contains(err.Error(), "trailing JSON value") {
				t.Fatalf("tail %q: %v", whitespace+tail, err)
			}
		}
	}
	contents := append(bytes.Clone(canonical), '[')
	contents = append(contents, bytes.Repeat([]byte("0,"), 1<<19)...)
	contents = append(contents, '0', ']')
	rejectionAllocations := func(input []byte) float64 {
		return testing.AllocsPerRun(1, func() {
			if _, err := Parse(input); err == nil || !strings.Contains(err.Error(), "trailing JSON value") {
				t.Fatalf("trailing array: %v", err)
			}
		})
	}
	small := rejectionAllocations(append(bytes.Clone(canonical), "[]"...))
	large := rejectionAllocations(contents)
	// Keep fixture parsing out of the comparison. A larger rejected suffix must
	// not add an interface node per number; allow a few allocator bookkeeping objects.
	if large > small+16 {
		t.Fatalf("array rejection allocations grew from %.0f to %.0f", small, large)
	}
}

func BenchmarkParseLockTrailingArray(b *testing.B) {
	for _, elements := range []int{1 << 14, 1 << 19} {
		b.Run(fmt.Sprint(elements), func(b *testing.B) {
			contents := append([]byte("{}["), bytes.Repeat([]byte("0,"), elements-1)...)
			contents = append(contents, '0', ']')
			b.ReportAllocs()
			b.ResetTimer()
			for range b.N {
				if _, err := Parse(contents); err == nil || !strings.Contains(err.Error(), "trailing JSON value") {
					b.Fatalf("trailing array: %v", err)
				}
			}
		})
	}
}
