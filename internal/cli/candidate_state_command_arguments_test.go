//go:build windows

package cli

import (
	"bufio"
	"bytes"
	"context"
	"crypto/sha256"
	"fmt"
	"io"
	"os/exec"
	"reflect"
	"strings"
	"testing"
	"time"
)

const stateCommandArgumentsProbe = `module main;
import module_state_command_arguments as arguments;
import module_state_command_options as options;
import std::bytes as bytes;
import std::vec as v;
import std::io as io;
@effects(console)fn void number(usize value){assert(io::write_usize(value).is_ok && io::write("\n").is_ok);}
@effects(console)fn void blob(ref u8[] raw){number(raw.len);assert(io::write_bytes(raw).is_ok && io::write("\n").is_ok);}
@effects(console)fn i32 main(){
 let input=switch(arguments::read()){case .ok(p):{p}case .err(e):{
  let code=arguments::exit_code(e);number(1);assert(io::write_i32(code).is_ok && io::write("\n").is_ok);
  switch(e){case .Invocation{.error=problem}:{number(switch(problem){case .Invalid:{1}case .UnsupportedGoMIR:{2}case .TooLarge:{3}case .Nul:{4}});}case .Host{.status=status,.index=index,.needed=needed}:{number(5);number(status as usize);number(index);number(needed);}case .Allocation:{number(6);}};
  return code;
 }};
 number(0);switch(arguments::parsed(ref input)){case .Help:{number(0);}case .Native{.options=chosen}:{number(chosen.action as usize);number(if(chosen.release){1}else{0});number(chosen.store_index);number(chosen.source_index);}};
 let argv=arguments::arguments(ref input);number(v::len<bytes::Buffer>(argv));var usize i=0;
 while(i<v::len<bytes::Buffer>(argv)){blob(bytes::view(v::at<bytes::Buffer>(argv,i)));i+=1;}return 0;
}`

func TestCandidateStateCommandArguments(t *testing.T) {
	t.Logf("actual-argv probe sha256=%x", sha256.Sum256([]byte(stateCommandArgumentsProbe)))
	type trial struct {
		name                     string
		args                     []string
		action, release, failure uint64
	}
	cases := []trial{
		{name: "help", args: []string{"--help"}},
		{name: "empty"},
		{name: "extra", args: []string{"--help", "extra"}},
		{name: "unknown", args: []string{"other"}, failure: 1},
		{name: "old-go-form", args: []string{"execute", "store", "source.fy"}, failure: 2},
	}
	for _, action := range []string{"init", "execute", "snapshot"} {
		for _, profile := range []string{"low", "optimized"} {
			a := uint64(1)
			if action == "execute" {
				a = 2
			}
			if action == "snapshot" {
				a = 3
			}
			p := uint64(0)
			if profile == "optimized" {
				p = 1
			}
			cases = append(cases, trial{action + "-" + profile, []string{action, "공백 있는 store 😀", `a\quoted folder\source.fy`, "--engine", "native-graph", "--profile", profile}, a, p, 0})
		}
	}
	valid := []string{"execute", "store", "source.fy", "--engine", "native-graph", "--profile", "low"}
	for _, tr := range []struct{ name, store, source string }{
		{"resize-127", strings.Repeat("x", 127), "source.fy"},
		{"resize-128", strings.Repeat("x", 128), "source.fy"},
		{"resize-129", strings.Repeat("x", 129), "source.fy"},
		{"long", strings.Repeat("가", 3000), "source.fy"},
		{"quoted", " tab\t newline\n\"trailing\\", "C:\\folder name\\source.fy"},
		{"literal-help", "--help", ".fy"},
	} {
		args := append([]string(nil), valid...)
		args[1], args[2] = tr.store, tr.source
		cases = append(cases, trial{tr.name, args, 2, 0, 0})
	}
	for i := range valid {
		args := append([]string(nil), valid...)
		args[i] = ""
		cases = append(cases, trial{fmt.Sprintf("empty-%d", i), args, 0, 0, 1})
	}
	for i := range cases {
		if cases[i].name == "empty" || cases[i].name == "extra" {
			cases[i].failure = 1
		}
	}
	candidateAtlasProbeSize(t, "module_state_command_arguments", stateCommandArgumentsProbe, 16<<10, func(t *testing.T, image string) {
		for _, tr := range cases {
			t.Run(tr.name, func(t *testing.T) {
				ctx, cancel := context.WithTimeout(t.Context(), 10*time.Second)
				defer cancel()
				cmd := exec.CommandContext(ctx, image, tr.args...)
				cmd.WaitDelay = time.Second
				var stdout, stderr bytes.Buffer
				cmd.Stdout, cmd.Stderr = &stdout, &stderr
				err := cmd.Run()
				code := 0
				if err != nil {
					if exit, ok := err.(*exec.ExitError); ok {
						code = exit.ExitCode()
					} else {
						t.Fatal(err)
					}
				}
				wantCode := 0
				if tr.failure != 0 {
					wantCode = 2
				}
				if code != wantCode || stderr.Len() != 0 {
					t.Fatalf("code=%d want=%d stderr=%q err=%v", code, wantCode, &stderr, err)
				}
				r := graphResponseReader{t, bufio.NewReader(&stdout)}
				if tr.failure != 0 {
					if r.number() != 1 || r.number() != 2 || r.number() != tr.failure {
						t.Fatal("wrong typed invocation failure")
					}
				} else {
					if r.number() != 0 || r.number() != tr.action {
						t.Fatal("wrong admitted action")
					}
					if tr.action != 0 && (r.number() != tr.release || r.number() != 1 || r.number() != 2) {
						t.Fatal("detached options/index mismatch")
					}
					if r.number() != uint64(len(tr.args)) {
						t.Fatal("argument count mismatch")
					}
					var got []string
					for range tr.args {
						got = append(got, string(r.blob()))
					}
					if !reflect.DeepEqual(got, tr.args) {
						t.Fatalf("exact argv changed: %#v != %#v", got, tr.args)
					}
				}
				if rest, err := io.ReadAll(r.r); err != nil || len(rest) != 0 {
					t.Fatalf("trailing output=%q err=%v", rest, err)
				}
			})
		}
	})
}
