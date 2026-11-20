// fairyc-c-interop runs the checked-in synchronous C interoperability example.
// It is a development harness, not a product export or generated-C interface.
package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/0disoft/fairyc/internal/cbackend"
	"github.com/0disoft/fairyc/internal/cdriver"
	"github.com/0disoft/fairyc/internal/compiler"
	"github.com/0disoft/fairyc/internal/cruntime"
	"github.com/0disoft/fairyc/internal/frontend"
	"github.com/0disoft/fairyc/internal/semantic"
	"github.com/0disoft/fairyc/internal/source"
	"github.com/0disoft/fairyc/internal/target"
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(args []string) error {
	flags := flag.NewFlagSet("fairyc-c-interop", flag.ContinueOnError)
	name := flags.String("profile", "low", "native profile: low or optimized")
	cc := flags.String("cc", "", "Linux C compiler: gcc or clang (required on Linux)")
	if err := flags.Parse(args); err != nil {
		if err == flag.ErrHelp {
			return nil
		}
		return err
	}
	if flags.NArg() != 0 {
		return fmt.Errorf("unexpected arguments; run from the repository root")
	}
	profile := cdriver.LowProfile
	if *name == "optimized" {
		profile = cdriver.OptimizedProfile
	} else if *name != "low" {
		return fmt.Errorf("unknown profile %q; choose low or optimized", *name)
	}
	if runtime.GOARCH != "amd64" {
		return fmt.Errorf("this example requires amd64, got %s/%s", runtime.GOOS, runtime.GOARCH)
	}
	emission, wantOutput := cbackend.Stage0Profile(), "accepted=2 callbacks=5\n"
	switch runtime.GOOS {
	case "windows":
		if *cc != "" {
			return fmt.Errorf("--cc is only supported on Linux; Windows uses MSVC")
		}
		emission, wantOutput = cbackend.Stage0WindowsMSVCProfile(), "accepted=2 callbacks=5\r\n"
	case "linux":
		if *cc != "gcc" && *cc != "clang" {
			return fmt.Errorf("Linux requires --cc gcc or --cc clang")
		}
	default:
		return fmt.Errorf("this example requires Windows or Linux, got %s", runtime.GOOS)
	}
	unit, companion, err := interopSources(emission)
	if err != nil {
		return err
	}
	result, err := buildAndRun(unit, companion, profile, *cc)
	if err != nil {
		return err
	}
	if result.exitCode != 0 || result.stdout != wantOutput || result.stderr != "" {
		return fmt.Errorf("interop failed: exit=%d stdout=%q stderr=%q", result.exitCode, result.stdout, result.stderr)
	}
	fmt.Print(result.stdout)
	fmt.Printf("interop ok: %s, %s, two separately compiled objects\n", profile.Name, result.toolchain)
	return nil
}

func interopSources(emission cbackend.Profile) (string, string, error) {
	root, err := filepath.Abs(filepath.Join("examples", "c_interop_callback"))
	if err != nil {
		return "", "", err
	}
	raw, err := os.ReadFile(filepath.Join(root, "main.fy"))
	if err != nil {
		return "", "", fmt.Errorf("read example (run from repository root): %w", err)
	}
	parsed, err := frontend.AnalyzeSyntax(source.NewFile("main.fy", raw), source.DefaultLimits())
	if err != nil {
		return "", "", fmt.Errorf("parse Fairy source: %w", err)
	}
	if parsed.AST == nil || len(parsed.Tree.Diagnostics) != 0 {
		return "", "", fmt.Errorf("Fairy syntax diagnostics: %+v", parsed.Tree.Diagnostics)
	}
	checked, err := compiler.Check([]semantic.File{{Path: "main.fy", Module: parsed.AST}})
	if err != nil {
		return "", "", fmt.Errorf("check Fairy source: %w", err)
	}
	if !checked.Verified {
		return "", "", fmt.Errorf("Fairy admission failed: diagnostics=%+v unsupported=%+v", checked.Semantic.Diagnostics, checked.Semantic.Unsupported)
	}
	generated, err := cbackend.Emit(checked.MIR, "main::main", emission)
	if err != nil {
		return "", "", err
	}
	rt, err := cruntime.ScalarSource(cruntime.NativeTrapExitBase)
	if err != nil {
		return "", "", err
	}
	// The compiler resolves legacy.h beside the original legacy.c.
	legacy := filepath.ToSlash(filepath.Join(root, "legacy.c"))
	if strings.ContainsAny(legacy, "\"\r\n") {
		return "", "", fmt.Errorf("example path cannot be used as a C include name")
	}
	return generated.Source, "#include \"" + legacy + "\"\n" + rt, nil
}

type interopOutcome struct {
	exitCode                  int32
	stdout, stderr, toolchain string
}

func buildAndRun(unit, companion string, profile cdriver.Profile, cc string) (interopOutcome, error) {
	if runtime.GOOS == "linux" {
		tc, err := cdriver.Resolve(target.X8664LinuxGNU, cc)
		if err != nil {
			return interopOutcome{}, err
		}
		artifact, err := cdriver.Build(tc, profile, unit, companion)
		if err != nil {
			return interopOutcome{}, err
		}
		result, err := cdriver.ExecuteLinuxTarget(artifact)
		return interopOutcome{result.ExitCode, result.Stdout, result.Stderr, tc.Name + " " + tc.Version}, err
	}
	tc, err := cdriver.ResolveWindowsHostMSVC()
	if err != nil {
		return interopOutcome{}, err
	}
	artifact, err := cdriver.BuildWindowsMSVCABIProbe(tc, profile, unit, companion)
	if err != nil {
		return interopOutcome{}, err
	}
	if len(artifact.Objects) != 2 {
		return interopOutcome{}, fmt.Errorf("expected separate Fairy and C objects, got %d", len(artifact.Objects))
	}
	result, err := cdriver.ExecuteWindowsMSVCABIProbe(artifact)
	return interopOutcome{result.ExitCode, result.Stdout, result.Stderr, "MSVC " + tc.Version}, err
}
