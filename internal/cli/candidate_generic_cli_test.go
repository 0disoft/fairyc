package cli

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

var genericCLIValues = []struct{ fairy, c, first, second string }{
	{"i8", "int8_t", "INT8_MIN", "INT8_MAX"},
	{"i16", "int16_t", "INT16_MIN", "INT16_MAX"},
	{"isize", "intptr_t", "INT64_C(-2147483649)", "INT64_C(4294967311)"},
	{"u16", "uint16_t", "0", "UINT16_MAX"},
	{"u64", "uint64_t", "UINT64_MAX", "0"},
	{"f32", "float", "-0.0f", "1.25f"},
	{"f64", "double", "-3.5", "0.0"},
}

const genericCLIPick = `fn T pick<T>(T first,T second){var T copy=first;copy=first;defer sink<T>(copy);return copy;}fn void sink<T>(T value){discard value;}`

func genericCLIWrappers(prefix string) string {
	var text strings.Builder
	for _, v := range genericCLIValues {
		// The third i32 argument distinguishes these private observer entries
		// from two-argument generic specializations without assuming frame IDs.
		fmt.Fprintf(&text, "pub fn %s probe_%s(%s first,%s second,i32 marker){return %spick<%s>(first,second);}", v.fairy, v.fairy, v.fairy, v.fairy, prefix, v.fairy)
	}
	return text.String()
}

// Retain the transport entries in the CLI's reachable C packet without adding
// literal construction or foreign aggregate admission. These source-only seeds
// are never executed. The C observer supplies real values to the retained entries.
func genericCLIRoot(prefix string, first, second int) string {
	var seeds, calls strings.Builder
	for _, v := range genericCLIValues {
		fmt.Fprintf(&seeds, "fn %s seed_%s(){return seed_%s();}", v.fairy, v.fairy, v.fairy)
		fmt.Fprintf(&calls, "discard probe_%s(seed_%s(),seed_%s(),0);", v.fairy, v.fairy, v.fairy)
	}
	return fmt.Sprintf("%sfn i32 main(){if(false){%s}return %spick<i32>(%d,%d);}", seeds.String(), calls.String(), prefix, first, second)
}

func genericCLIWorkspace(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	writeTestSource(t, root, "fairy.toml", `schema = "fairyc.manifest/v1"
[package]
uid = "test.generic-cli"
name = "generic-cli"
version = "1.0.0"
language = "fairyc-language/20"
source = "src"
`)
	writeTestSource(t, root, "src/main.fy", "module main;"+genericCLIPick+genericCLIWrappers("")+genericCLIRoot("", 42, 43))
	writeTestSource(t, root, "tests/generics.fy", "module tests::generics;"+genericCLIPick+genericCLIWrappers("")+`@test fn void transport(){assert(pick<i32>(42,43)==42);assert(pick<bool>(true,false));}`)
	if err := os.Mkdir(filepath.Join(root, "cache"), 0700); err != nil {
		t.Fatal(err)
	}
	return root
}

func TestCandidateGenericTransportCLI(t *testing.T) {
	t.Setenv("FAIRYC_CLOSURE_CLI_PROFILE", os.Getenv("FAIRYC_GENERIC_CLI_PROFILE"))
	for _, profile := range copiedClosureCLIProfiles(t) {
		t.Run(profile.Name, func(t *testing.T) {
			image := copiedClosureCLIHost(t, profile)
			installGenericCLIPacketCapture(t, image)
			root := genericCLIWorkspace(t)
			packet := filepath.Join(root, "observed.bin")
			t.Setenv("FAIRYC_GENERIC_CLI_PACKET", packet)
			command := copiedClosureCommand{t: t, image: image, root: root}
			command.cache(0, "cache: cold stored", "check")
			command.cache(0, "cache: hit", "check", "--locked")
			for _, release := range []bool{false, true} {
				args := []string{"build", "--locked"}
				if release {
					args = append(args, "--release")
				}
				command.cache(0, "cache: hit", args...)
				observeGenericCLIPacket(t, packet, false, 42)
				ctx, cancel := context.WithTimeout(t.Context(), time.Minute)
				out, err := exec.CommandContext(ctx, filepath.Join(root, "build", "generic-cli.exe")).CombinedOutput()
				cancel()
				if e, ok := err.(*exec.ExitError); !ok || e.ExitCode() != 42 || len(out) != 0 {
					t.Fatalf("generated generic executable: %v %s", err, out)
				}
				args[0] = "run"
				command.cache(42, "cache: hit", args...)
			}
			command.run(0, "test", "--list", "--json", "--locked")
			for _, flags := range [][]string{nil, {"--native"}, {"--native", "--release"}} {
				out, _ := command.run(0, append([]string{"test", "--json", "--locked"}, flags...)...)
				if !strings.Contains(out, `"passed":1`) {
					t.Fatal(out)
				}
			}
			t.Logf("%d independent CLI processes; seven actual scalar observations per product profile", command.calls)
		})
	}
}
