package windowsarchiveprobe

import (
	"strings"
	"testing"
)

func TestGenerateIsDeterministicAndClosesArchiveSelectionInputs(t *testing.T) {
	first := Generate()
	second := Generate()
	if first != second {
		t.Fatalf("Generate() is not deterministic")
	}
	if first.ArchiveName != "choice.lib" || first.SelectedMember != "chosen.obj" ||
		first.UnusedMember != "unused.obj" || first.DirectObject != "direct.obj" ||
		first.ArchiveCandidate == first.DirectCandidate || first.ExpectedExit != 0 {
		t.Fatalf("archive probe identity = %+v", first)
	}
	checks := []struct {
		text string
		want string
	}{
		{first.CallerSource, `#pragma section(".fysel", read)`},
		{first.CallerSource, "fy_archive_target"},
		{first.CallerSource, "&fy_pick_any_value"},
		{first.ChosenSource, "__declspec(selectany)"},
		{first.ChosenSource, "= 17"},
		{first.UnusedSource, "fy_archive_unused"},
		{first.DirectSource, "__declspec(selectany)"},
		{first.DirectSource, "= 23"},
	}
	for _, check := range checks {
		if !strings.Contains(check.text, check.want) {
			t.Fatalf("generated source does not contain %q", check.want)
		}
	}
}
