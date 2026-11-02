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

func staticOperationCLIProvider() string {
	var source strings.Builder
	source.WriteString("@interface pub fn T pick<T>(T first,T second){}")
	for _, scalar := range []string{"i32", "i64", "u8", "u32", "usize", "bool", "i8", "i16", "isize", "u16", "u64", "f32", "f64"} {
		fmt.Fprintf(&source, "@implements(pick,%s) fn %s select_%s(%s first,%s second){return first;}", scalar, scalar, scalar, scalar, scalar)
	}
	source.WriteString(`@interface pub fn T seed<T>(){}
@implements(seed,i32) fn i32 make32(){return 42;}
@implements(seed,i64) fn i64 make64(){return 64;}
@interface pub fn T four<T>(T a,T b,T c,T d){}
@implements(four,i32) fn i32 four32(i32 a,i32 b,i32 c,i32 d){return b;}
fn i32 spare(){return 8;}`)
	return source.String()
}

func staticOperationCLIChecks(prefix string) string {
	return strings.ReplaceAll(`assert(OPpick<i64>(40,90)==40);assert(OPpick<u8>(4,8)==4);assert(OPpick<u32>(6,9)==6);assert(OPpick<usize>(7,10)==7);assert(OPpick<bool>(true,false));assert(OPseed<i32>()==42);assert(OPfour<i32>(1,42,3,4)==42);`, "OP", prefix)
}

func staticOperationCLIMain(prefix string) string {
	root := genericCLIRoot(prefix, 42, 43)
	return genericCLIWrappers(prefix) + strings.Replace(root, "return "+prefix+"pick<i32>", staticOperationCLIChecks(prefix)+"return "+prefix+"pick<i32>", 1)
}

func staticOperationCLIWorkspace(t *testing.T) string {
	t.Helper()
	root := genericCLIWorkspace(t)
	writeTestSource(t, root, "src/main.fy", "module main;"+staticOperationCLIProvider()+staticOperationCLIMain(""))
	writeTestSource(t, root, "tests/generics.fy", "module tests::generics;"+staticOperationCLIProvider()+"@test fn void selected(){"+staticOperationCLIChecks("")+"assert(pick<i32>(21,99)==21);}")
	return root
}

func TestCandidateStaticOperationCLI(t *testing.T) {
	t.Setenv("FAIRYC_CLOSURE_CLI_PROFILE", os.Getenv("FAIRYC_STATIC_CLI_PROFILE"))
	for _, profile := range copiedClosureCLIProfiles(t) {
		t.Run(profile.Name, func(t *testing.T) {
			image := copiedClosureCLIHost(t, profile)
			installGenericCLIPacketCapture(t, image)
			root := staticOperationCLIWorkspace(t)
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
					t.Fatalf("generated static-operation executable: %v %s", err, out)
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
			t.Logf("%d independent CLI processes; thirteen scalar providers and zero/four-argument operations", command.calls)
		})
	}
}
