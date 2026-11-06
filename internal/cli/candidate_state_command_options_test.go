package cli

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// Binary stdin transports arbitrary argv bytes to a PURE parser. NUL and large
// arguments must never be sent to exec.Command as OS command-line arguments.
// This envelope is a test fixture, not a new command or production entry point.
const candidateStateCommandOptionsProbe = `module main;
import module_state_command_options as options;
import std::bytes as bytes; import std::vec as v; import std::io as io;
enum Tested { Accepted { options::Parsed parsed; }; Rejected { options::InvocationError error; }; }
fn usize number(ref u8[] raw,ref mut usize at){
 let offset=*at;let n=(raw[offset] as usize)*16777216+(raw[offset+1] as usize)*65536+(raw[offset+2] as usize)*256+(raw[offset+3] as usize);*at+=4;return n;
}
@effects(console) fn i32 main(){
 var input=bytes::empty();var u8[4096] block=[0;4096];
 while(true){let n=switch(io::read_stdin(ref mut block)){case .ok(n):{n}case .err(_):{return 90;}};
  if(n==0){break;}let ref u8[] raw=ref block;
  if(n>2097152-bytes::len(ref input) || !bytes::append(ref mut input,raw[..n]).is_ok){return 91;}}
 let raw=bytes::view(ref input);if(raw.len<4){return 92;}var usize at=0;let count=number(raw,ref mut at);
 if(count>1024){return 93;}var v::Vec<bytes::Buffer> argv=v::empty<bytes::Buffer>();var usize i=0;
 while(i<count){if(raw.len-at<4){return 94;}let size=number(raw,ref mut at);if(size>raw.len-at){return 95;}
  let data=switch(bytes::copy(raw[at..at+size])){case .ok(p):{p}case .err(_):{return 96;}};
  if(!v::push<bytes::Buffer>(ref mut argv,data).is_ok){return 97;}at+=size;i+=1;}
 if(at!=raw.len){return 98;}
 let Tested result=switch(options::parse(ref argv)){case .ok(parsed):{.Accepted{.parsed}}case .err(error):{.Rejected{.error}}};
 // Verify every byte and length on success AND failure, then destroy argv
 // before using the result: no mutation, normalization, or borrowed result.
 at=4;i=0;if(v::len<bytes::Buffer>(ref argv)!=count){return 99;}
 while(i<count){let size=number(raw,ref mut at);
  if(!bytes::equal(bytes::view(v::at<bytes::Buffer>(ref argv,i)),raw[at..at+size])){return 100;}at+=size;i+=1;}
 discard v::clear<bytes::Buffer>(ref mut argv);
 switch(result){
  case .Rejected{.error=error}:{
   switch(error){case .Invalid:{discard io::write("invalid");}case .UnsupportedGoMIR:{discard io::write("unsupported-go-mir");}case .TooLarge:{discard io::write("too-large");}case .Nul:{discard io::write("nul");}};
   return options::exit_code(error);
  }
  case .Accepted{.parsed=parsed}:{switch(parsed){
   case .Help:{discard io::write("help");return 0;}
   case .Native{.options=p}:{
    let copied=p;discard io::write_u32(copied.action);discard io::write(if(copied.release){":1:"}else{":0:"});
    discard io::write_usize(copied.store_index);discard io::write(":");discard io::write_usize(copied.source_index);return 0;
   }
  };}
 };return 101;
}`

type candidateStateOptionsTrial struct {
	name string
	args []string
	want string
	code int
}

func candidateStateOptionsTrials() []candidateStateOptionsTrial {
	var trials []candidateStateOptionsTrial
	add := func(name string, args []string, want string, code int) {
		trials = append(trials, candidateStateOptionsTrial{name, append([]string{}, args...), want, code})
	}
	base := []string{"execute", "store", "source.fy", "--engine", "native-graph", "--profile", "low"}
	change := func(index int, value string) []string {
		out := append([]string{}, base...)
		out[index] = value
		return out
	}
	for i, action := range []string{"init", "execute", "snapshot"} {
		for _, profile := range []string{"low", "optimized"} {
			args := change(0, action)
			args[6] = profile
			release := 0
			if profile == "optimized" {
				release = 1
			}
			add(action+"/"+profile, args, fmt.Sprintf("%d:%d:1:2", i+1, release), 0)
		}
	}
	// Explicit expectations, authored independently of the Go predicate below.
	for i, source := range []string{".fy", "dir/.fy", `dir\.fy`, `C:\dir\unit.fy`, "a.b.fy", "dir.fy/child.fy", `dir.fy\child.fy`, "../x.fy", "한글/소스.fy", " unit.fy", "a\nb.fy", "raw-\xff.fy"} {
		add(fmt.Sprintf("source-accepted-%d", i), change(2, source), "2:0:1:2", 0)
	}
	for i, source := range []string{"", "unit.FY", "unit.Fy", "unit.fy ", "unit.fy.", "unit.fy/", `unit.fy\`, "unit.fy/child", `unit.fy\child`, "unit.fy:stream", "noext", ".", "..", "unit.fy\n", "unit.fy/.."} {
		add(fmt.Sprintf("source-rejected-%d", i), change(2, source), "invalid", 2)
	}
	// Store is only required to be nonempty; retain exact caller bytes, including
	// whitespace, flag-like spelling, dot components and nonexistent paths.
	for i, store := range []string{" ", "--help", "--engine", "../missing", `C:\a b\store`, "\n\t"} {
		add(fmt.Sprintf("store-accepted-%d", i), change(1, store), "2:0:1:2", 0)
	}
	for index := range base {
		add(fmt.Sprintf("empty-%d", index), change(index, ""), "invalid", 2)
		add(fmt.Sprintf("nul-%d", index), change(index, base[index]+"\x00"), "nul", 2)
		want, code := "invalid", 2
		if index == 1 {
			want, code = "2:0:1:2", 0
		}
		add(fmt.Sprintf("help-position-%d", index), change(index, "--help"), want, code)
	}
	add("nul-source-before-suffix", change(2, "source\x00.fy"), "nul", 2)
	add("nul-store-leading", change(1, "\x00store"), "nul", 2)
	for i, replacement := range []struct {
		index int
		value string
	}{
		{0, "Execute"}, {0, "run"}, {0, " execute"}, {3, "--engine=native-graph"}, {3, "--ENGINE"},
		{4, "go-mir"}, {4, "Native-graph"}, {5, "--profile=low"}, {6, "release"}, {6, "LOW"}, {6, "Optimized"}, {6, " low"}, {6, "low "},
	} {
		add(fmt.Sprintf("wrong-token-%d", i), change(replacement.index, replacement.value), "invalid", 2)
	}
	add("help", []string{"--help"}, "help", 0)
	add("help-with-nul", []string{"--help\x00"}, "nul", 2)
	add("help-wrong-case", []string{"--Help"}, "invalid", 2)
	add("help-extra", []string{"--help", "extra"}, "invalid", 2)
	add("help-after", append(append([]string{}, base...), "--help"), "invalid", 2)
	add("help-before", append([]string{"--help"}, base...), "invalid", 2)
	add("empty-argv", nil, "invalid", 2)
	for n := 2; n <= 9; n++ {
		if n == 3 || n == 7 {
			continue
		}
		args := append([]string{}, base...)
		for len(args) < n {
			args = append(args, "extra")
		}
		add(fmt.Sprintf("arity-%d", n), args[:n], "invalid", 2)
	}
	for _, action := range []string{"init", "execute", "snapshot"} {
		add("go-default-"+action, []string{action, "store", "source.fy"}, "unsupported-go-mir", 2)
	}
	add("three-invalid", []string{"garbage", "", ""}, "unsupported-go-mir", 2)
	add("three-nul", []string{"init", "store\x00", "source.fy"}, "nul", 2)
	add("reordered-flags", []string{"execute", "store", "source.fy", "--profile", "low", "--engine", "native-graph"}, "invalid", 2)
	for index := 0; index < len(base)-1; index++ {
		args := append([]string{}, base...)
		args[index], args[index+1] = args[index+1], args[index]
		want, code := "invalid", 2
		// Swapping store/source yields source "store", so this also rejects.
		add(fmt.Sprintf("adjacent-swap-%d", index), args, want, code)
	}
	for _, extra := range []string{"--target", "--profile", "--engine", "low", ""} {
		add("extra-"+extra, append(append([]string{}, base...), extra), "invalid", 2)
	}
	// Exactly 1 MiB aggregate argv bytes is admitted, one more is rejected.
	other := 0
	for i, s := range base {
		if i != 1 {
			other += len(s)
		}
	}
	add("large-within", change(1, strings.Repeat("s", 65536)), "2:0:1:2", 0)
	add("limit-exact", change(1, strings.Repeat("s", (1<<20)-other)), "2:0:1:2", 0)
	add("limit-over", change(1, strings.Repeat("s", (1<<20)-other+1)), "too-large", 2)
	add("long-source", change(2, strings.Repeat("p", 65536)+".fy"), "2:0:1:2", 0)
	add("long-source-invalid", change(2, strings.Repeat("p", 65536)+".FY"), "invalid", 2)
	add("large-help", []string{strings.Repeat("h", 1<<20)}, "invalid", 2)
	add("oversize-help", []string{strings.Repeat("h", (1<<20)+1)}, "too-large", 2)
	add("oversize-go-default", []string{"init", strings.Repeat("s", 1<<20), "source.fy"}, "too-large", 2)
	add("limit-before-nul", change(1, strings.Repeat("s", (1<<20)-other)+"\x00"), "too-large", 2)
	add("arity-before-nul", []string{"--help", "\x00"}, "invalid", 2)
	add("too-many", make([]string, 1024), "invalid", 2)
	return trials
}

// Independent syntax-only Go reference to main.go/runNative's invocation gate.
// Never call runNative: source/store I/O and transaction execution are not parser
// tests. Deliberate private admission differences are checked separately.
func candidateStateOptionsGoGate(args []string) bool {
	return len(args) == 1 && args[0] == "--help" || len(args) == 7 &&
		(args[0] == "init" || args[0] == "execute" || args[0] == "snapshot") &&
		args[1] != "" && filepath.Ext(args[2]) == ".fy" && args[3] == "--engine" &&
		args[4] == "native-graph" && args[5] == "--profile" && (args[6] == "low" || args[6] == "optimized")
}

func TestCandidateStateCommandOptions(t *testing.T) {
	if runtime.GOOS != "windows" || runtime.GOARCH != "amd64" {
		t.Skip("Windows path semantics/native profiles; Linux execution is deferred")
	}
	trials := candidateStateOptionsTrials()
	for _, trial := range trials {
		if trial.want == "nul" || trial.want == "too-large" {
			continue
		}
		if got := candidateStateOptionsGoGate(trial.args); got != (trial.code == 0) {
			t.Fatalf("independent matrix %s conflicts with Go syntax gate: %v", trial.name, got)
		}
	}
	for _, path := range []string{
		"examples/lexer_candidate/src/module_state_command_options.fy",
		"tools/dev/fairyc-state/main.go", "tools/dev/fairyc-state/native.go",
		"docs/adr/0500-local-contract-state-execution-and-persistence.md",
	} {
		data, err := os.ReadFile(filepath.Join("../..", path))
		if err != nil {
			t.Fatal(err)
		}
		t.Logf("source SHA-256 %s %x", path, sha256.Sum256(data))
	}
	candidateAtlasProbeSize(t, "module_state_command_options", candidateStateCommandOptionsProbe, 16<<10, func(t *testing.T, image string) {
		for _, trial := range trials {
			t.Run(trial.name, func(t *testing.T) {
				var input bytes.Buffer
				if err := binary.Write(&input, binary.BigEndian, uint32(len(trial.args))); err != nil {
					t.Fatal(err)
				}
				for _, arg := range trial.args {
					if err := binary.Write(&input, binary.BigEndian, uint32(len(arg))); err != nil {
						t.Fatal(err)
					}
					input.WriteString(arg)
				}
				code, out := candidateAtlasProcess(t, image, input.Bytes())
				if code != trial.code || out != trial.want {
					t.Fatalf("native exit=%d output=%q; want %d %q", code, out, trial.code, trial.want)
				}
			})
		}
		t.Logf("%d pure-parser native observations; argv preserved and destroyed before result use", len(trials))
	})
}
