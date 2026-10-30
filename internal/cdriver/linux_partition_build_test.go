package cdriver

import (
	"errors"
	"runtime"
	"testing"

	"github.com/0disoft/fairyc/internal/target"
)

func TestLinuxPartitionedBuild(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("requires native Linux GCC/Clang")
	}
	parts := CPartitions{Header: "extern const unsigned char shared[2];\nint helper(void);\n", Sources: []string{
		"#include \"fy_bootstrap.h\"\nint helper(void) { return shared[0]; }\n",
		"#include \"fy_bootstrap.h\"\nconst unsigned char shared[2] = {23, 0};\nint main(void) { return helper(); }\n",
	}}
	for _, name := range []string{"gcc", "clang"} {
		t.Run(name, func(t *testing.T) {
			cc, err := Resolve(target.X8664LinuxGNU, name)
			if err != nil {
				t.Fatal(err)
			}
			for _, profile := range []Profile{LowProfile, OptimizedProfile} {
				t.Run(profile.Name, func(t *testing.T) {
					artifact, err := BuildLinuxPartitions(cc, profile, parts, "typedef int runtime_translation_unit;\n")
					if err != nil {
						t.Fatal(err)
					}
					stats := artifact.PartitionBuild
					if stats == nil || stats.Workers != linuxCompileWorkers(stats.GOMAXPROCS, stats.AvailableMemoryBytes) {
						t.Fatalf("missing or inconsistent compiler admission: %+v", stats)
					}
					t.Logf("partition resources: %+v", stats)
					outcome, err := ExecuteLinuxTarget(artifact)
					if err != nil || outcome.ExitCode != 23 || outcome.Stdout != "" || outcome.Stderr != "" {
						t.Fatalf("partitioned native result: %+v %v", outcome, err)
					}
				})
			}
			broken := CPartitions{Header: parts.Header, Sources: append([]string(nil), parts.Sources...)}
			broken.Sources[0] = "int broken( {"
			artifact, err := BuildLinuxPartitions(cc, LowProfile, broken, "typedef int runtime_translation_unit;\n")
			if err == nil || len(artifact.Bytes) != 0 {
				t.Fatal("failed partition build published an image")
			}
			var failure *PartitionBuildError
			if !errors.As(err, &failure) || failure.Stats.Workers < 1 || failure.Stats.Workers > 2 {
				t.Fatalf("failed build lost worker admission: %v", err)
			}
		})
	}
}
