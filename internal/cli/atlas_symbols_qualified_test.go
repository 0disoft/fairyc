package cli

import (
	"strings"
	"testing"
)

func TestExperimentalAtlasSymbolsQualifiedNamesAreExact(t *testing.T) {
	root := writeWorkspaceSessionFixture(t, `module main;
struct Cell{i32 value;}struct Other{i32 value;}
fn i32 Other::get(ref Other this){return this.value;}
fn i32 Cell::get(ref Cell this){return this.value;}
fn i32 main(){return 42;}`)
	for _, version := range []string{"1", "2", "3"} {
		for _, name := range []string{"Cell::get", "Other::get", "get", "Cell::absent", "cell::get"} {
			args := append(atlasSymbolsArguments("example.session.app"), "--catalog-version", version, "--name", name)
			code, output, diagnostic := runForTest(append([]string{"experimental", "atlas-symbols"}, args...), root)
			if code != ExitSuccess || diagnostic != "" {
				t.Fatalf("catalog %s name %q: %d %s", version, name, code, diagnostic)
			}
			got := decodeAtlasSymbols(t, output).Catalog
			want := 0
			if name == "Cell::get" || name == "Other::get" {
				want = 1
			}
			if !got.Complete || got.Name != name || got.Total != want || len(got.Symbols) != want || want == 1 && got.Symbols[0].Name != name {
				t.Fatalf("catalog %s exact qualified filter: %+v", version, got)
			}
		}
	}
}

func TestAtlasSymbolsQualifiedNameValidation(t *testing.T) {
	for _, name := range []string{"Cell::get", "get", strings.Repeat("a", 507) + "::get"} {
		if _, err := parseAtlasSymbolsOptions(append(atlasSymbolsArguments("example.session.app")[1:], "--name", name)); err != nil {
			t.Fatalf("valid qualified path %q: %v", name, err)
		}
	}
	for _, name := range []string{"", "::get", "Cell::", "Cell:::get", "Cell:get", "Cell/get", "Cell::9get", "Cell::get\x00", "Cell::get\n", strings.Repeat("a", 508) + "::get"} {
		if _, err := parseAtlasSymbolsOptions(append(atlasSymbolsArguments("example.session.app")[1:], "--name", name)); err == nil {
			t.Fatalf("malformed or oversized qualified path accepted: %q", name)
		}
	}
}
