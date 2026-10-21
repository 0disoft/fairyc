package cdriver

import (
	"fmt"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/cruntime"
	"github.com/0disoft/fairyc/internal/target"
)

func TestNativeTargetSelection(t *testing.T) {
	for _, test := range []struct {
		os, arch string
		want     target.ID
	}{
		{"windows", "amd64", target.X8664WindowsMSVC}, {"linux", "amd64", target.X8664LinuxGNU},
		{"darwin", "amd64", ""}, {"linux", "arm64", ""},
	} {
		got, err := hostNativeTarget(test.os, test.arch)
		if got != test.want || (err != nil) != (test.want == "") {
			t.Fatalf("%s/%s = %q, %v", test.os, test.arch, got, err)
		}
	}
	first, _ := LookupNativeTarget(target.X8664LinuxGNU)
	first.Compilers[0] = "changed"
	second, _ := LookupNativeTarget(target.X8664LinuxGNU)
	if second.Compilers[0] != "gcc" {
		t.Fatal("shared compiler metadata changed")
	}
	if _, err := ResolveNativeToolchain("missing", ""); err == nil {
		t.Fatal("unknown native target accepted")
	}
	if err := validateNativeArtifact(NativeArtifact{Artifact: Artifact{Toolchain: Toolchain{Target: target.X8664LinuxGNU}}}, target.X8664WindowsMSVC); err == nil {
		t.Fatal("foreign target artifact accepted")
	}
}

func TestNativeOutcomeReceipts(t *testing.T) {
	for code := uint8(1); code <= 7; code++ {
		exit := int32(NativeTrapExitBase) + int32(code)
		normal, err := classifyNativeOutcome("test", exit, "out", "err", "")
		if err != nil || normal.TrapCategory != "" {
			t.Fatalf("reserved ordinary result classified as trap: %+v %v", normal, err)
		}
		trapped, err := classifyNativeOutcome("test", exit, "out", "err", string([]byte{code}))
		category, _ := NativeTrapCategory(code)
		if err != nil || trapped.TrapCategory != category || trapped.Stdout != "out" || trapped.Stderr != "err" {
			t.Fatalf("trap %d: %+v, %v", code, trapped, err)
		}
		for _, receipt := range []string{string([]byte{0}), string([]byte{code, code}), string([]byte{code + 1})} {
			if _, err := classifyNativeOutcome("test", exit, "", "", receipt); err == nil {
				t.Fatalf("invalid receipt %q accepted", receipt)
			}
		}
	}
	if _, err := classifyNativeOutcome("test", 42, "", "", "\x03"); err == nil {
		t.Fatal("unexpected receipt accepted")
	}
}

func TestNativeRejectOutput(t *testing.T) {
	for _, stream := range []struct{ stdout, stderr string }{{stdout: "out"}, {stderr: "err"}} {
		if _, err := finishNativeOutcome("test", 0, stream.stdout, stream.stderr, "", NativeExecutionOptions{RejectOutput: true}); err == nil {
			t.Fatal("unexpected output accepted")
		}
	}
	outcome, err := finishNativeOutcome("test", NativeTrapExitBase+6, "", "", "\x06", NativeExecutionOptions{RejectOutput: true})
	if err != nil || outcome.TrapCategory != "assertion-failed" {
		t.Fatalf("silent trap: %+v %v", outcome, err)
	}
}

func TestNativeToolchainBuildExecute(t *testing.T) {
	id, err := HostNativeTarget()
	if err != nil {
		t.Skip(err)
	}
	toolchain, err := ResolveNativeToolchain(id, "")
	if err != nil {
		t.Fatal(err)
	}
	sourceRuntime := cruntime.LinuxRunSource
	if id == target.X8664WindowsMSVC {
		sourceRuntime = cruntime.WindowsRunSource
	}
	support, err := sourceRuntime(NativeTrapExitBase)
	if err != nil {
		t.Fatal(err)
	}
	for _, profile := range []Profile{LowProfile, OptimizedProfile} {
		for _, trap := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/trap=%t", profile.Name, trap), func(t *testing.T) {
				body := "return 131;"
				if trap {
					body = "fy_runtime_trap_at_v0(3, \"src/main.fy\", 11, UINT64_C(17), UINT64_C(29));"
				}
				source := "#include <stdint.h>\n#include <stdio.h>\n_Noreturn void fy_runtime_trap_at_v0(uint32_t, const char *, uint32_t, uint64_t, uint64_t);\nint main(int argc, char **argv) { if(argc != 2 || getchar() != 'A') return 42; (void)fputs(argv[1], stdout); (void)fflush(stdout); (void)fputs(\"warning\", stderr); " + body + " }\n"
				artifact, err := toolchain.Build(NativeBuildOptions{Profile: profile, UnitSource: source, RuntimeSource: support, TrapControl: true})
				if err != nil {
					t.Fatal(err)
				}
				outcome, err := toolchain.Execute(artifact, NativeExecutionOptions{Input: strings.NewReader("A"), Arguments: []string{"argument"}})
				category := ""
				if trap {
					category = "division-by-zero"
				}
				if err != nil || outcome.ExitCode != 131 || outcome.Stdout != "argument" || outcome.Stderr != "warning" || outcome.TrapCategory != category {
					t.Fatalf("%+v %v", outcome, err)
				}
				if trap {
					if outcome.TrapLocation == nil || *outcome.TrapLocation != (TrapLocation{Source: "src/main.fy", StartByte: 17, EndByte: 29}) {
						t.Fatalf("native trap location: %+v", outcome.TrapLocation)
					}
				} else if outcome.TrapLocation != nil {
					t.Fatal("ordinary return acquired a trap location")
				}
			})
		}
	}
}
