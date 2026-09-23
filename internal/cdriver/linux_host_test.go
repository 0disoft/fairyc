package cdriver

import (
	"runtime"
	"testing"

	"github.com/0disoft/fairyc/internal/cruntime"
	"github.com/0disoft/fairyc/internal/target"
)

func TestExecuteLinuxTargetSeparatesNormalReturnFromSameTrapStatus(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("requires Linux amd64 glibc and GCC")
	}
	toolchain, err := Resolve(target.X8664LinuxGNU, "gcc")
	if err != nil {
		t.Fatal(err)
	}
	support, err := cruntime.LinuxRunSource(128)
	if err != nil {
		t.Fatal(err)
	}
	for _, test := range []struct {
		name     string
		source   string
		stdout   string
		category uint8
	}{
		{name: "ordinary", source: "#include <stdio.h>\nint main(void) { fputs(\"ordinary\", stdout); return 131; }\n", stdout: "ordinary"},
		{name: "trap", source: "#include <stdint.h>\n_Noreturn void fy_runtime_trap(uint32_t);\nint main(void) { fy_runtime_trap(3); }\n", category: 3},
	} {
		t.Run(test.name, func(t *testing.T) {
			artifact, err := Build(toolchain, LowProfile, test.source, support)
			if err != nil {
				t.Fatal(err)
			}
			outcome, err := ExecuteLinuxTarget(artifact)
			if err != nil {
				t.Fatal(err)
			}
			if outcome.ExitCode != 131 || outcome.TrapCategory != test.category || outcome.Stdout != test.stdout || outcome.Stderr != "" {
				t.Fatalf("outcome = %+v", outcome)
			}
		})
	}
}

func TestExecuteLinuxTargetRejectsUnsupportedHostOrArtifact(t *testing.T) {
	_, err := ExecuteLinuxTarget(Artifact{})
	if err == nil {
		t.Fatal("empty Linux artifact was accepted")
	}
	if runtime.GOOS == "linux" && runtime.GOARCH == "amd64" {
		_, err = ExecuteLinuxTarget(Artifact{Toolchain: Toolchain{Target: target.X8664LinuxGNU}, ConsumerMode: C11ConsumerMode, Bytes: []byte("not ELF")})
		if err == nil {
			t.Fatal("non-ELF Linux artifact was accepted")
		}
	}
}
