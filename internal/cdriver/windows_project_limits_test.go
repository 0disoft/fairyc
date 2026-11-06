package cdriver

import (
	"runtime"
	"strings"
	"testing"
	"time"
)

func TestWindowsProjectHeaderRoles(t *testing.T) {
	buffer := strings.Repeat("h", (8<<20)+1)
	for _, compilerHost := range []bool{false, true} {
		for _, size := range []int{0, (2 << 20) - 1, 2 << 20, (2 << 20) + 1, 2099985, (8 << 20) - 1, 8 << 20, (8 << 20) + 1} {
			limit := 2 << 20
			if compilerHost {
				limit = 8 << 20
			}
			err := validateWindowsProjectInputs(buffer[:size], "x", []string{"x"}, "x", compilerHost)
			if (err == nil) != (size > 0 && size <= limit) {
				t.Fatalf("compilerHost=%t header=%d: %v", compilerHost, size, err)
			}
		}
	}
}

func TestWindowsProjectSharedBounds(t *testing.T) {
	buffer := strings.Repeat("x", (8<<20)+1)
	for _, compilerHost := range []bool{false, true} {
		for _, field := range []string{"literals", "runtime", "body", "parts"} {
			limit := 2 << 20
			if field == "runtime" {
				limit = 8 << 20
			} else if field == "parts" {
				limit = 4096
			}
			for _, size := range []int{0, limit, limit + 1} {
				literals, support, units := "x", "x", []string{"x"}
				switch field {
				case "literals":
					literals = buffer[:size]
				case "runtime":
					support = buffer[:size]
				case "body":
					units[0] = buffer[:size]
				case "parts":
					units = make([]string, size)
					for i := range units {
						units[i] = "x"
					}
				}
				err := validateWindowsProjectInputs("x", literals, units, support, compilerHost)
				if (err == nil) != (size > 0 && size <= limit) {
					t.Fatalf("compilerHost=%t %s=%d: %v", compilerHost, field, size, err)
				}
			}
		}
		// Shared strings keep the test allocation below 9MiB while exercising
		// exact 96MiB transport accounting over all logical source bytes.
		units := make([]string, 48)
		for i := range units {
			units[i] = buffer[:2<<20]
		}
		units[47] = buffer[:(2<<20)-3]
		if err := validateWindowsProjectInputs("x", "x", units, "x", compilerHost); err != nil {
			t.Fatal("exact total rejected", compilerHost, err)
		}
		units[47] = buffer[:(2<<20)-2]
		if err := validateWindowsProjectInputs("x", "x", units, "x", compilerHost); err == nil || err.Error() != "project C transport exceeds 96 MiB" {
			t.Fatal("one-over total accepted", compilerHost, err)
		}
	}
}

func TestWindowsCompilerHostLargeHeader(t *testing.T) {
	if runtime.GOOS != "windows" || runtime.GOARCH != "amd64" {
		t.Skip("requires Windows amd64 MSVC")
	}
	tc, err := ResolveWindowsHostMSVC()
	if err != nil {
		t.Fatal(err)
	}
	const measuredHeaderBytes = 2099985
	const padding = "/* private compiler host */\n"
	header := strings.Repeat(padding, measuredHeaderBytes/len(padding)) + strings.Repeat(" ", measuredHeaderBytes%len(padding))
	literals := "#include \"fy_project.h\"\nint compiler_literal=0;\n"
	units := []string{"#include \"fy_project.h\"\nint main(void){return 42;}\n"}
	support := "int compiler_runtime=0;\n"
	for _, profile := range []Profile{LowProfile, OptimizedProfile} {
		t.Run(profile.Name, func(t *testing.T) {
			ordinary, err := BuildWindowsMSVCTargetPartsWithStackReserve(tc, profile, header, literals, units, support, time.Minute, 16<<20)
			if err == nil || err.Error() != "invalid bounded project C inputs" || len(ordinary.Bytes) != 0 || len(ordinary.Objects) != 0 {
				t.Fatal("ordinary admission widened", err)
			}
			private, err := BuildWindowsMSVCCompilerHostPartsWithStackReserve(tc, profile, header, literals, units, support, time.Minute, 16<<20)
			if err != nil || len(private.Bytes) == 0 || len(private.Objects) != 0 {
				t.Fatal("private image admission/ownership", err)
			}
			out, err := ExecuteWindowsMSVCTarget(private)
			if err != nil || out.ExitCode != 42 || out.Stdout != "" || out.Stderr != "" {
				t.Fatalf("private compiler-host image: %+v %v", out, err)
			}
		})
	}
}
