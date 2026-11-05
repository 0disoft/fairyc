package cli

import (
	"encoding/json"
	"strconv"
	"strings"
	"testing"
)

const candidateCallsOptionsMain = `module main;
import project_calls_options as options;import project_atlas_write as w;
import std::bytes as bytes;import std::io as io;
@effects(console) fn i32 main(){
 let p=switch(options::arguments()){case .ok(p):{p}case .err(_):{return 2;}};var out=bytes::empty();var usize meter=0;
 if(!w::raw(ref mut out,b"{",ref mut meter).is_ok || !w::pair(ref mut out,b"uid",options::uid(ref p),ref mut meter).is_ok || !w::raw(ref mut out,b",",ref mut meter).is_ok || !w::pair(ref mut out,b"symbol",options::symbol(ref p),ref mut meter).is_ok || !w::raw(ref mut out,b",",ref mut meter).is_ok || !w::pair(ref mut out,b"direction",options::direction(ref p),ref mut meter).is_ok){return 4;}
 if(!w::raw(ref mut out,b",\"packages\":",ref mut meter).is_ok || !w::number(ref mut out,options::packages(ref p),ref mut meter).is_ok || !w::raw(ref mut out,b",\"calls\":",ref mut meter).is_ok || !w::number(ref mut out,options::calls(ref p),ref mut meter).is_ok || !w::raw(ref mut out,b",\"output\":",ref mut meter).is_ok || !w::number(ref mut out,options::output(ref p),ref mut meter).is_ok || !w::raw(ref mut out,b",\"version\":",ref mut meter).is_ok || !w::number(ref mut out,options::version(ref p) as usize,ref mut meter).is_ok || !w::raw(ref mut out,b",\"map\":",ref mut meter).is_ok || !w::number(ref mut out,options::map_version(ref p) as usize,ref mut meter).is_ok){return 4;}
 if(!w::raw(ref mut out,if(options::locked(ref p)){b",\"locked\":true"}else{b",\"locked\":false"},ref mut meter).is_ok || !w::raw(ref mut out,if(options::outgoing(ref p)){b",\"outgoing\":true"}else{b",\"outgoing\":false"},ref mut meter).is_ok || !w::raw(ref mut out,if(options::incoming(ref p)){b",\"incoming\":true}"}else{b",\"incoming\":false}"},ref mut meter).is_ok){return 4;}
 return if(io::write_bytes(bytes::view(ref out)).is_ok){0}else{4};
}`

func TestCandidateCallsOptions(t *testing.T) {
	base := []string{"--package-uid", "example.session.app", "--symbol", "fysym1-sha256-128-" + strings.Repeat("a", 32), "--direction", "both", "--max-packages", "8", "--max-calls", "128", "--max-output-bytes", "1048576"}
	add := func(extra ...string) []string { return append(append([]string{}, base...), extra...) }
	cases := [][]string{base, add("--locked"), add("--catalog-version", "0"), add("--catalog-version", "1"), add("--catalog-version", "2", "--locked"), {}, {"--help"}, add("--unknown", "1"), add("--locked", "--locked"), add("--catalog-version", "00"), add("--catalog-version", "3"), add("--catalog-version", "1", "--catalog-version", "2"), add("extra"), add("--symbol=abc"), add("--catalog-version"), add("--catalog-version", "--locked")}
	for i := 0; i < len(base); i += 2 {
		cases = append(cases, append(append([]string{}, base[:i]...), base[i+2:]...), add(base[i], base[i+1]))
		values := []string{"", "-1", "+1", "01", "0", "1", "65535", "65536", "65537", "1073741823", "1073741824", "1073741825", "18446744073709551616"}
		switch base[i] {
		case "--package-uid":
			values = []string{"one", "Example.app", "a.b", "a." + strings.Repeat("b", 254), "a." + strings.Repeat("b", 255)}
		case "--symbol":
			values = []string{"", "g0_" + strings.Repeat("a", 64), "fysym1-sha256-128-" + strings.Repeat("a", 31), "fysym1-sha256-128-" + strings.Repeat("a", 33), "fysym1-sha256-256-" + strings.Repeat("a", 64), "fysym1-sha256-128-" + strings.Repeat("A", 32), "fysym1-sha256-256-" + strings.Repeat("A", 64), "fysym1-sha256-256-" + strings.Repeat("0", 64)}
		case "--direction":
			values = []string{"", "outgoing", "incoming", "both", "Outgoing", "out", "--locked"}
		}
		for _, value := range values {
			args := append([]string{}, base...)
			args[i+1] = value
			cases = append(cases, args)
		}
	}
	t.Logf("Go/Fairy call argument cases: %d per profile", len(cases))
	candidateAtlasProbeSize(t, "project_calls_options", candidateCallsOptionsMain, 32<<10, func(t *testing.T, image string) {
		for _, args := range cases {
			want, err := parseAtlasCallsOptions(args)
			code, output := candidateAtlasProcess(t, image, nil, append([]string{"experimental", "atlas-calls", "list"}, args...)...)
			if err != nil {
				if code != ExitInvalidInvocation || output != "" {
					t.Fatalf("invalid %q: %d %q; Go %v", args, code, output, err)
				}
				continue
			}
			var got struct {
				UID, Symbol, Direction     string
				Packages, Calls, Output    uint64
				Version, Map               int
				Locked, Outgoing, Incoming bool
			}
			version := 0
			if want.catalogVersion != "" {
				version, _ = strconv.Atoi(want.catalogVersion)
			}
			if code != 0 || json.Unmarshal([]byte(output), &got) != nil || got.UID != want.packageUID || got.Symbol != want.symbol || got.Direction != string(want.direction) || got.Packages != uint64(want.maxPackages) || got.Calls != uint64(want.maxCalls) || got.Output != want.maxOutputBytes || got.Version != version || got.Map != version+1 || got.Locked != want.locked || got.Outgoing != (want.direction != "incoming") || got.Incoming != (want.direction != "outgoing") {
				t.Fatalf("valid %q: %d %q; want %+v", args, code, output, want)
			}
		}
		for _, action := range []string{"", "show", "List"} {
			code, output := candidateAtlasProcess(t, image, nil, append([]string{"experimental", "atlas-calls", action}, base...)...)
			if code != ExitInvalidInvocation || output != "" {
				t.Fatalf("invalid action %q: %d %q", action, code, output)
			}
		}
	}, "project_atlas_write")
}
