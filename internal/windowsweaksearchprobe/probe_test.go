package windowsweaksearchprobe

import (
	"strings"
	"testing"
)

func TestGenerateIsDeterministicAndSeparatesArchiveMembers(t *testing.T) {
	t.Parallel()
	first := Generate()
	second := Generate()
	if first != second {
		t.Fatal("Generate() is not deterministic")
	}
	if first.ObjectName != "weakmode.obj" || first.ArchiveName != "weakmode.lib" ||
		first.NoLibraryMember == first.LibraryMember || first.NoLibraryWeak == first.LibraryWeak || first.ExpectedExit != 0 {
		t.Fatalf("weak-search identities = %+v", first)
	}
	for _, value := range []struct {
		source, symbol string
		count          int
	}{
		{first.CallerSource, first.NoLibraryEntry, 2}, {first.CallerSource, first.LibraryEntry, 2},
		{first.NoLibrarySource, first.NoLibraryWeak, 1}, {first.NoLibrarySource, first.NoLibrarySentinel, 1},
		{first.LibrarySource, first.LibraryWeak, 1},
	} {
		if strings.Count(value.source, value.symbol) != value.count {
			t.Fatalf("source contains %q %d times, want %d", value.symbol, strings.Count(value.source, value.symbol), value.count)
		}
	}
}
