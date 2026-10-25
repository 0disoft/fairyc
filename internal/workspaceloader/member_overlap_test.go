package workspaceloader

import (
	"errors"
	"fmt"
	"strings"
	"testing"
)

func TestWorkspaceMemberOverlapKeepsFirstConflict(t *testing.T) {
	for _, test := range []struct {
		name                   string
		members                []string
		reason, path, previous string
	}{
		{"case", []string{"Packages/A", "packages/a"}, ReasonPathCase, "packages/a", "Packages/A"},
		{"parent first", []string{"A", "a/B"}, ReasonOverlappingMember, "a/B", "A"},
		{"child first", []string{"a/B", "A"}, ReasonOverlappingMember, "A", "a/B"},
		{"earliest child", []string{"a/Z", "a/B", "A"}, ReasonOverlappingMember, "A", "a/Z"},
		{"first failing member", []string{"A/Z", "b", "a", "B"}, ReasonOverlappingMember, "a", "A/Z"},
		{"unicode case", []string{"Σ/A", "σ/a"}, ReasonPathCase, "σ/a", "Σ/A"},
	} {
		t.Run(test.name, func(t *testing.T) {
			var failure *LoadError
			if err := rejectOverlappingMembers(test.members); !errors.As(err, &failure) || failure.Reason != test.reason || failure.Path != test.path || !strings.Contains(failure.Detail, fmt.Sprintf("%q", test.previous)) {
				t.Fatalf("member conflict: %v, want %s %q against %q", err, test.reason, test.path, test.previous)
			}
		})
	}
}

func TestWorkspaceMemberOverlapRespectsComponentBoundaries(t *testing.T) {
	for _, members := range [][]string{
		{"a", "ab", "a-b", "a_b", "a.b"},
		{"Packages/A/B", "packages/a/C", "packages/ab/B"},
		{},
	} {
		if err := rejectOverlappingMembers(members); err != nil {
			t.Fatalf("nonoverlapping members %v: %v", members, err)
		}
	}
}

func BenchmarkWorkspaceMemberOverlap(b *testing.B) {
	for _, count := range []int{256, MaxPackages - 1} {
		b.Run(fmt.Sprint(count), func(b *testing.B) {
			members := make([]string, count)
			prefix := "Packages/" + strings.Repeat("Compiler", 12) + "/"
			for index := range members {
				members[index] = fmt.Sprintf("%sPkg%04d", prefix, index)
			}
			b.ReportAllocs()
			b.ResetTimer()
			for range b.N {
				if err := rejectOverlappingMembers(members); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}
