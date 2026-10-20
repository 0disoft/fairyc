package main

import (
	"strings"
	"testing"
)

func TestSearchLineSemantics(t *testing.T) {
	for _, tc := range []struct {
		input, prefix string
	}{
		{"", "lines=0 matches=0 "},
		{"\n\n", "lines=2 matches=0 "},
		{"errorerror\r\nERROR\n\x00error", "lines=3 matches=2 "},
		{"err\nor\n", "lines=2 matches=0 "},
		{strings.Repeat("x", 4094) + "error", "lines=1 matches=1 "},
		{strings.Repeat("x", 9000) + "error\n", "lines=1 matches=1 "},
	} {
		if got := searchReference([]byte(tc.input), 0); !strings.HasPrefix(got, tc.prefix) {
			t.Fatalf("input length %d: got %q, want prefix %q", len(tc.input), got, tc.prefix)
		}
	}
	if len(searchInput()) != 1<<20 {
		t.Fatal("file corpus must be exactly 1 MiB")
	}
}

func TestSearchRevisionIdentity(t *testing.T) {
	for _, language := range []string{"fairy", "zig", "rust", "c"} {
		seen := map[string]bool{}
		for _, revision := range []int{0, 1, 5, 10} {
			hash, _, _ := sourceIdentity(searchSources(language, revision))
			if seen[hash] {
				t.Fatalf("%s revision %d did not change source", language, revision)
			}
			seen[hash] = true
		}
	}
	if searchReference(searchInput(), 0) == searchReference(searchInput(), 5) {
		t.Fatal("search edits must change verified output")
	}
}
