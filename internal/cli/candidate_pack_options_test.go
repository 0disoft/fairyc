package cli

import (
	"encoding/json"
	"strconv"
	"strings"
	"testing"
)

const candidatePackOptionsMain = `module main;
import project_pack_options as options;import project_atlas_write as w;
import std::bytes as bytes;import std::io as io;
@effects(console) fn i32 main(){
 let p=switch(options::arguments()){case .ok(p):{p}case .err(_):{return 2;}};
 var out=bytes::empty();var usize meter=0;
 if(!w::raw(ref mut out,b"{",ref mut meter).is_ok || !w::pair(ref mut out,b"action",options::action(ref p),ref mut meter).is_ok || !w::raw(ref mut out,b",",ref mut meter).is_ok || !w::pair(ref mut out,b"uid",options::uid(ref p),ref mut meter).is_ok || !w::raw(ref mut out,b",",ref mut meter).is_ok || !w::pair(ref mut out,b"symbol",options::symbol(ref p),ref mut meter).is_ok || !w::raw(ref mut out,b",",ref mut meter).is_ok || !w::pair(ref mut out,b"direction",options::direction(ref p),ref mut meter).is_ok){return 4;}
 if(!w::raw(ref mut out,b",\"records\":",ref mut meter).is_ok || !w::number(ref mut out,options::records(ref p),ref mut meter).is_ok || !w::raw(ref mut out,b",\"relations\":",ref mut meter).is_ok || !w::number(ref mut out,options::relations(ref p),ref mut meter).is_ok || !w::raw(ref mut out,b",\"depth\":",ref mut meter).is_ok || !w::number(ref mut out,options::depth(ref p),ref mut meter).is_ok){return 4;}
 if(!w::raw(ref mut out,b",\"selection\":",ref mut meter).is_ok || !w::number(ref mut out,options::selection(ref p),ref mut meter).is_ok || !w::raw(ref mut out,b",\"output\":",ref mut meter).is_ok || !w::number(ref mut out,options::output(ref p),ref mut meter).is_ok || !w::raw(ref mut out,b",\"version\":",ref mut meter).is_ok || !w::number(ref mut out,options::version(ref p) as usize,ref mut meter).is_ok){return 4;}
 if(!w::raw(ref mut out,if(options::locked(ref p)){b",\"locked\":true"}else{b",\"locked\":false"},ref mut meter).is_ok || !w::raw(ref mut out,b",\"selected\":[",ref mut meter).is_ok){return 4;}
 var usize i=0;var bool first=true;
 while(i<10){if(options::has_relation(ref p,i)){if(!first && !w::raw(ref mut out,b",",ref mut meter).is_ok || !w::string(ref mut out,options::relation(i),ref mut meter).is_ok){return 4;}first=false;}i+=1;}
 if(!w::raw(ref mut out,b"]}",ref mut meter).is_ok){return 4;}
 return if(io::write_bytes(bytes::view(ref out)).is_ok){0}else{4};
}`

type candidatePackOptionsCase struct {
	name   string
	action string
	args   []string
}

type candidatePackOptionsWant struct {
	Action, UID, Symbol, Direction string
	Records, Relations, Depth      uint64
	Selection, Output              uint64
	Version                        uint32
	Locked                         bool
	Selected                       []string `json:"selected"`
}

func candidatePackUIDs() (string, string, string) {
	return "example.session.app", "fysym1-sha256-128-" + strings.Repeat("a", 32), "fysym1-sha256-256-" + strings.Repeat("b", 64)
}

func candidatePackRelationKinds() []string {
	return []string{"allocates", "borrows-mutable", "borrows-shared", "calls", "calls-foreign", "drops", "may-trap", "moves", "returns-error", "uses-unsafe"}
}

func candidatePackBase(action string) []string {
	uid, visible, _ := candidatePackUIDs()
	switch action {
	case "overview":
		return []string{"--package-uid", uid, "--max-records", "64", "--max-selection-bytes", "65536", "--max-output-bytes", "131072"}
	case "show":
		return []string{"--package-uid", uid, "--symbol", visible, "--direction", "outgoing", "--relation", "calls", "--max-records", "64", "--max-relations", "8", "--max-selection-bytes", "65536", "--max-output-bytes", "131072"}
	case "around":
		return []string{"--package-uid", uid, "--symbol", visible, "--direction", "outgoing", "--relation", "calls", "--max-records", "64", "--max-relations", "8", "--max-depth", "2", "--max-selection-bytes", "65536", "--max-output-bytes", "131072"}
	default:
		panic("unknown candidate pack action " + action)
	}
}

func candidatePackClone(values []string) []string { return append([]string{}, values...) }

func candidatePackReplace(args []string, flag, value string) []string {
	out := candidatePackClone(args)
	for i := 0; i+1 < len(out); i += 2 {
		if out[i] == flag {
			out[i+1] = value
			return out
		}
	}
	return append(out, flag, value)
}

func candidatePackWithout(args []string, flag string) []string {
	out := make([]string, 0, len(args))
	for i := 0; i < len(args); i++ {
		if args[i] != flag {
			out = append(out, args[i])
			continue
		}
		if flag != "--locked" && i+1 < len(args) {
			i++
		}
	}
	return out
}

func candidatePackOptionsCases() []candidatePackOptionsCase {
	uid, _, full := candidatePackUIDs()
	var cases []candidatePackOptionsCase
	add := func(name, action string, args []string) {
		cases = append(cases, candidatePackOptionsCase{name: name, action: action, args: candidatePackClone(args)})
	}
	for _, action := range []string{"overview", "show", "around"} {
		base := candidatePackBase(action)
		add(action+"/default-version", action, base)
		for _, version := range []string{"1", "2", "3"} {
			add(action+"/version-"+version, action, append(candidatePackClone(base), "--pack-version", version))
		}
		add(action+"/locked", action, append(candidatePackClone(base), "--locked"))
		for _, version := range []string{"", "0", "4", "01", "2.0", "--locked"} {
			add(action+"/bad-version-"+version, action, append(candidatePackClone(base), "--pack-version", version))
		}
		add(action+"/duplicate-version", action, append(candidatePackClone(base), "--pack-version", "1", "--pack-version", "2"))
		add(action+"/missing-version-value", action, append(candidatePackClone(base), "--pack-version"))
		add(action+"/duplicate-locked", action, append(candidatePackClone(base), "--locked", "--locked"))
		add(action+"/unknown-flag", action, append(candidatePackClone(base), "--walk", "1"))
		add(action+"/unknown-bare-flag", action, append(candidatePackClone(base), "--future"))
	}

	for _, action := range []string{"show", "around"} {
		base := candidatePackBase(action)
		for _, direction := range []string{"outgoing", "incoming", "both"} {
			add(action+"/direction-"+direction, action, candidatePackReplace(base, "--direction", direction))
		}
		for _, direction := range []string{"", "Outgoing", "out", "bidirectional"} {
			add(action+"/bad-direction-"+direction, action, candidatePackReplace(base, "--direction", direction))
		}
		for _, relation := range candidatePackRelationKinds() {
			args := base
			if relation != "calls" {
				args = append(candidatePackClone(base), "--relation", relation)
			}
			add(action+"/relation-"+relation, action, args)
		}
		for _, relation := range []string{"", "Calls", "call", "calls_foreign", "allocates ", "walk"} {
			add(action+"/bad-relation-"+relation, action, append(candidatePackClone(base), "--relation", relation))
		}
		add(action+"/duplicate-relation", action, append(candidatePackClone(base), "--relation", "calls"))
		add(action+"/missing-relation", action, candidatePackWithout(base, "--relation"))
	}

	positiveValues := []string{"", "-1", "+1", "01", "0", "1", "65535", "65536", "65537", "1073741823", "1073741824", "1073741825", "18446744073709551616"}
	for _, action := range []string{"overview", "show", "around"} {
		base := candidatePackBase(action)
		flags := []string{"--max-records", "--max-selection-bytes", "--max-output-bytes"}
		if action != "overview" {
			flags = append(flags, "--max-relations")
		}
		for _, flag := range flags {
			for _, value := range positiveValues {
				add(action+"/"+flag+"-"+value, action, candidatePackReplace(base, flag, value))
			}
		}
	}
	for _, value := range []string{"", "-1", "+1", "01", "0", "1", "63", "64", "65", "18446744073709551616"} {
		add("around/--max-depth-"+value, "around", candidatePackReplace(candidatePackBase("around"), "--max-depth", value))
	}

	for _, action := range []string{"overview", "show", "around"} {
		base := candidatePackBase(action)
		for _, flag := range []string{"--package-uid", "--max-records", "--max-selection-bytes", "--max-output-bytes"} {
			add(action+"/missing-"+flag, action, candidatePackWithout(base, flag))
		}
		if action != "overview" {
			for _, flag := range []string{"--symbol", "--direction", "--relation", "--max-relations"} {
				add(action+"/missing-"+flag, action, candidatePackWithout(base, flag))
			}
		}
		if action == "around" {
			add(action+"/missing---max-depth", action, candidatePackWithout(base, "--max-depth"))
		}
		add(action+"/duplicate-package-uid", action, append(candidatePackClone(base), "--package-uid", uid))
		for _, flag := range []string{"--max-records", "--max-selection-bytes", "--max-output-bytes"} {
			value := base[len(base)-1]
			for i := 0; i < len(base); i += 2 {
				if base[i] == flag {
					value = base[i+1]
				}
			}
			add(action+"/duplicate-"+flag, action, append(candidatePackClone(base), flag, value))
		}
	}
	for _, action := range []string{"show", "around"} {
		base := candidatePackBase(action)
		add(action+"/duplicate-symbol", action, append(candidatePackClone(base), "--symbol", full))
		add(action+"/duplicate-direction", action, append(candidatePackClone(base), "--direction", "both"))
		add(action+"/duplicate-max-relations", action, append(candidatePackClone(base), "--max-relations", "9"))
	}
	add("around/duplicate-max-depth", "around", append(candidatePackClone(candidatePackBase("around")), "--max-depth", "3"))

	overview := candidatePackBase("overview")
	for _, extra := range [][]string{
		{"--symbol", full}, {"--direction", "both"}, {"--relation", "calls"},
		{"--max-relations", "1"}, {"--max-depth", "0"},
	} {
		add("overview/rejects-"+extra[0], "overview", append(candidatePackClone(overview), extra...))
	}
	add("show/rejects-depth", "show", append(candidatePackClone(candidatePackBase("show")), "--max-depth", "0"))

	uidValues := []string{"", "one", "Example.app", "a.b", "a." + strings.Repeat("b", 254), "a." + strings.Repeat("b", 255), "a..b", "a.b-", "a.b_c"}
	for _, value := range uidValues {
		add("around/uid-"+value, "around", candidatePackReplace(candidatePackBase("around"), "--package-uid", value))
	}
	symbolValues := []string{
		"", "g0_" + strings.Repeat("a", 64),
		"fysym1-sha256-128-" + strings.Repeat("a", 31),
		"fysym1-sha256-128-" + strings.Repeat("a", 33),
		"fysym1-sha256-256-" + strings.Repeat("a", 63),
		"fysym1-sha256-256-" + strings.Repeat("a", 65),
		"fysym1-sha256-128-" + strings.Repeat("A", 32),
		"fysym1-sha256-256-" + strings.Repeat("A", 64),
		"fysym1-sha256-128-00000000000000000000000000000000",
	}
	for _, value := range symbolValues {
		add("around/symbol-"+strconv.Itoa(len(value)), "around", candidatePackReplace(candidatePackBase("around"), "--symbol", value))
	}

	add("around/permuted", "around", []string{
		"--max-output-bytes", "777", "--relation", "moves", "--max-depth", "3",
		"--package-uid", uid, "--locked", "--max-relations", "12", "--direction", "both",
		"--max-selection-bytes", "999", "--symbol", full, "--max-records", "21",
		"--relation", "calls", "--pack-version", "3",
	})

	fullRelations := candidatePackRelationKinds()
	maxArgs := []string{
		"--package-uid", uid, "--symbol", full, "--direction", "both",
		"--max-records", "65536", "--max-relations", "65536", "--max-depth", "64",
		"--max-selection-bytes", "1073741824", "--max-output-bytes", "1073741824",
		"--locked", "--pack-version", "3",
	}
	for i := len(fullRelations) - 1; i >= 0; i-- {
		maxArgs = append(maxArgs, "--relation", fullRelations[i])
	}
	add("around/all-ten-maximum-argv", "around", maxArgs)
	return cases
}

func candidatePackOptionsOracle(action string, args []string) (candidatePackOptionsWant, error) {
	want := candidatePackOptionsWant{Action: action, Version: 1, Selected: []string{}}
	switch action {
	case "overview":
		got, err := parseAtlasPackOverviewOptions(args)
		if err != nil {
			return candidatePackOptionsWant{}, err
		}
		want.UID = got.packageUID
		want.Records = uint64(got.maxRecords)
		want.Selection = got.maxSelectionBytes
		want.Output = got.maxOutputBytes
		want.Locked = got.locked
		if got.packVersion != "" {
			version, _ := strconv.ParseUint(got.packVersion, 10, 32)
			want.Version = uint32(version)
		}
		return want, nil
	case "show", "around":
		got, err := parseAtlasPackRelationOptions(action, args)
		if err != nil {
			return candidatePackOptionsWant{}, err
		}
		want.UID, want.Symbol, want.Direction = got.packageUID, got.symbol, string(got.direction)
		want.Records, want.Relations, want.Depth = uint64(got.maxRecords), uint64(got.maxRelations), uint64(got.maxDepth)
		want.Selection, want.Output = got.maxSelectionBytes, got.maxOutputBytes
		want.Locked = got.locked
		if got.packVersion != "" {
			version, _ := strconv.ParseUint(got.packVersion, 10, 32)
			want.Version = uint32(version)
		}
		for _, relation := range got.relations {
			want.Selected = append(want.Selected, string(relation))
		}
		return want, nil
	default:
		panic("unknown oracle action " + action)
	}
}

func TestCandidatePackOptions(t *testing.T) {
	cases := candidatePackOptionsCases()
	if len(cases) == 0 {
		t.Fatal("empty parser case set")
	}
	for _, c := range cases {
		if len(c.args)+4 != 43 {
			continue
		}
		if c.name != "around/all-ten-maximum-argv" {
			t.Fatalf("unexpected 43-argv case %q", c.name)
		}
	}
	t.Logf("Go/Fairy Atlas pack argument cases: %d per profile", len(cases))
	candidateAtlasProbeSize(t, "project_pack_options", candidatePackOptionsMain, 32<<10, func(t *testing.T, image string) {
		observations := 0
		for _, c := range cases {
			want, oracleErr := candidatePackOptionsOracle(c.action, c.args)
			code, output := candidateAtlasProcess(t, image, nil, append([]string{"experimental", "atlas-pack", c.action}, c.args...)...)
			observations++
			if oracleErr != nil {
				if code != ExitInvalidInvocation || output != "" {
					t.Fatalf("%s: invalid args %q: code=%d output=%q; Go=%v", c.name, c.args, code, output, oracleErr)
				}
				continue
			}
			var got candidatePackOptionsWant
			if code != ExitSuccess || json.Unmarshal([]byte(output), &got) != nil {
				t.Fatalf("%s: valid args %q: code=%d output=%q; want=%+v", c.name, c.args, code, output, want)
			}
			if got.Action != want.Action || got.UID != want.UID || got.Symbol != want.Symbol || got.Direction != want.Direction || got.Records != want.Records || got.Relations != want.Relations || got.Depth != want.Depth || got.Selection != want.Selection || got.Output != want.Output || got.Version != want.Version || got.Locked != want.Locked || strings.Join(got.Selected, ",") != strings.Join(want.Selected, ",") {
				t.Fatalf("%s: args %q\n got=%+v\nwant=%+v", c.name, c.args, got, want)
			}
		}
		for _, action := range []string{"", "walk", "List"} {
			code, output := candidateAtlasProcess(t, image, nil, "experimental", "atlas-pack", action)
			observations++
			if code != ExitInvalidInvocation || output != "" {
				t.Fatalf("invalid query %q: code=%d output=%q", action, code, output)
			}
		}
		t.Logf("%d independent Atlas pack parser-native observations", observations)
	}, "project_atlas_write")
}
