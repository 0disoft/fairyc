package cli

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/semanticsnapshot"
)

const candidateCatalogOptionsMain = `module main;
import project_catalog_options as options;import project_atlas_write as w;
import std::bytes as bytes;import std::io as io;
@effects(console) fn i32 main(){
 let p=switch(options::arguments()){case .ok(p):{p}case .err(_):{return 2;}};
 var out=bytes::empty();var usize meter=0;
 if(!w::raw(ref mut out,b"{",ref mut meter).is_ok || !w::pair(ref mut out,b"uid",options::uid(ref p),ref mut meter).is_ok || !w::raw(ref mut out,b",",ref mut meter).is_ok || !w::pair(ref mut out,b"module",options::module_name(ref p),ref mut meter).is_ok || !w::raw(ref mut out,b",",ref mut meter).is_ok || !w::pair(ref mut out,b"name",options::name(ref p),ref mut meter).is_ok || !w::raw(ref mut out,b",",ref mut meter).is_ok || !w::pair(ref mut out,b"after",options::after(ref p),ref mut meter).is_ok || !w::raw(ref mut out,b",\"kinds\":[",ref mut meter).is_ok){return 4;}
 var usize i=0;var bool first=true;while(i<5){if(options::has_kind(ref p,i)){if(!first && !w::raw(ref mut out,b",",ref mut meter).is_ok){return 4;}if(!w::string(ref mut out,options::kind(i),ref mut meter).is_ok){return 4;}first=false;}i+=1;}
 if(!w::raw(ref mut out,b"],\"symbols\":",ref mut meter).is_ok || !w::number(ref mut out,options::symbols(ref p),ref mut meter).is_ok || !w::raw(ref mut out,b",\"selection\":",ref mut meter).is_ok || !w::number(ref mut out,options::selection(ref p),ref mut meter).is_ok || !w::raw(ref mut out,b",\"output\":",ref mut meter).is_ok || !w::number(ref mut out,options::output(ref p),ref mut meter).is_ok || !w::raw(ref mut out,b",\"version\":",ref mut meter).is_ok || !w::number(ref mut out,options::version(ref p) as usize,ref mut meter).is_ok || !w::raw(ref mut out,if(options::locked(ref p)){b",\"locked\":true}"}else{b",\"locked\":false}"},ref mut meter).is_ok){return 4;}
 return if(io::write_bytes(bytes::view(ref out)).is_ok){0}else{4};
}`

func TestCandidateCatalogOptions(t *testing.T) {
	base := []string{"--package-uid", "example.session.app", "--kind", "function", "--max-symbols", "16", "--max-selection-bytes", "65536", "--max-output-bytes", "131072"}
	add := func(extra ...string) []string { return append(append([]string{}, base...), extra...) }
	cases := [][]string{base, add("--locked"), add("--catalog-version", "1"), add("--catalog-version", "2"), add("--catalog-version", "3"), add("--name", "Cell::get"), add("--module", "not an identifier"), add("--kind", "struct", "--kind", "enum", "--kind", "resource", "--kind", "foreign-function"), add("--after", "fysym1-sha256-256-"+strings.Repeat("a", 64)), {}, {"--help"}, add("--unknown", "1"), add("--locked", "--locked"), add("--kind", "function"), add("--kind", "constant"), add("--catalog-version", "01"), add("--catalog-version", "4"), add("--catalog-version", "1", "--catalog-version", "2"), add("--after", "fysym1-sha256-128-"+strings.Repeat("a", 32)), add("--after", "fysym1-sha256-256-"+strings.Repeat("A", 64)), add("--name", "A:::get"), add("--name", "::get"), add("--name", "get::"), add("--name", ""), add("--module", ""), add("extra"), add("--name=leaf"), add("--module"), add("--kind", "--locked")}
	for i := 0; i < len(base); i += 2 {
		cases = append(cases, append(append([]string{}, base[:i]...), base[i+2:]...), add(base[i], base[i+1]))
		values := []string{"", "-1", "+1", "01", "0", "1", "65535", "65536", "65537", "1073741823", "1073741824", "1073741825", "18446744073709551616"}
		if i == 0 {
			values = []string{"one", "Example.app", "a.b", "a." + strings.Repeat("b", 254), "a." + strings.Repeat("b", 255)}
		}
		for _, value := range values {
			args := append([]string{}, base...)
			args[i+1] = value
			cases = append(cases, args)
		}
	}
	for _, flag := range []string{"--name", "--module"} {
		cases = append(cases, add(flag, "--locked"))
		for _, n := range []int{511, 512, 513} {
			cases = append(cases, add(flag, strings.Repeat("a", n)))
		}
	}
	all := add("--catalog-version", "3", "--module", "main", "--name", "get", "--after", "fysym1-sha256-256-"+strings.Repeat("a", 64), "--kind", "struct", "--kind", "enum", "--kind", "resource", "--kind", "foreign-function", "--locked")
	cases = append(cases, all)
	t.Logf("Go/Fairy argument cases: %d per profile", len(cases))
	candidateAtlasProbeSize(t, "project_catalog_options", candidateCatalogOptionsMain, 32<<10, func(t *testing.T, image string) {
		for _, args := range cases {
			want, err := parseAtlasSymbolsOptions(args)
			code, output := candidateAtlasProcess(t, image, nil, append([]string{"experimental", "atlas-symbols", "list"}, args...)...)
			if err != nil {
				if code != ExitInvalidInvocation || output != "" {
					t.Fatalf("invalid options %q: code=%d output=%q, Go=%v", args, code, output, err)
				}
				continue
			}
			var got struct {
				UID, Module, Name, After   string
				Kinds                      []semanticsnapshot.DeclarationKind
				Symbols, Selection, Output uint64
				Version                    int
				Locked                     bool
			}
			version := 1
			if want.catalogVersion == "2" {
				version = 2
			} else if want.catalogVersion == "3" {
				version = 3
			}
			kinds, _ := json.Marshal(want.kinds)
			if code != ExitSuccess || json.Unmarshal([]byte(output), &got) != nil {
				t.Fatalf("valid options %q: code=%d output=%q", args, code, output)
			}
			gotKinds, _ := json.Marshal(got.Kinds)
			if got.UID != want.packageUID || got.Module != want.module || got.Name != want.name || got.After != want.after || string(gotKinds) != string(kinds) || got.Symbols != uint64(want.maxSymbols) || got.Selection != want.maxSelectionBytes || got.Output != want.maxOutputBytes || got.Version != version || got.Locked != want.locked {
				t.Fatalf("options drift %q: got=%+v want=%+v", args, got, want)
			}
		}
		for _, action := range []string{"", "show", "List"} {
			code, output := candidateAtlasProcess(t, image, nil, append([]string{"experimental", "atlas-symbols", action}, base...)...)
			if code != ExitInvalidInvocation || output != "" {
				t.Fatalf("invalid action %q: %d %q", action, code, output)
			}
		}
	}, "project_atlas_write")
}
