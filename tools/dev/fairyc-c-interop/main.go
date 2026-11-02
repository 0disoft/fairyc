// fairyc-c-interop runs the checked-in synchronous C interoperability example.
// It is a development harness, not a product export or generated-C interface.
package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"github.com/0disoft/fairyc/internal/cbackend"
	"github.com/0disoft/fairyc/internal/cdriver"
	"github.com/0disoft/fairyc/internal/compiler"
	"github.com/0disoft/fairyc/internal/cruntime"
	"github.com/0disoft/fairyc/internal/frontend"
	"github.com/0disoft/fairyc/internal/semantic"
	"github.com/0disoft/fairyc/internal/source"
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
	if runtime.GOOS != "windows" || runtime.GOARCH != "amd64" {
		return fmt.Errorf("this example harness currently requires Windows amd64 with MSVC; Linux replay is deferred")
	}
	root, err := filepath.Abs(filepath.Join("examples", "c_interop_callback"))
	if err != nil {
		return err
	}
	raw, err := os.ReadFile(filepath.Join(root, "main.fy"))
	if err != nil {
		return fmt.Errorf("read example (run from repository root): %w", err)
	}
	parsed, err := frontend.AnalyzeSyntax(source.NewFile("main.fy", raw), source.DefaultLimits())
	if err != nil {
		return fmt.Errorf("parse Fairy source: %w", err)
	}
	if parsed.AST == nil || len(parsed.Tree.Diagnostics) != 0 {
		return fmt.Errorf("Fairy syntax diagnostics: %+v", parsed.Tree.Diagnostics)
	}
	checked, err := compiler.Check([]semantic.File{{Path: "main.fy", Module: parsed.AST}})
	if err != nil {
		return fmt.Errorf("check Fairy source: %w", err)
	}
	if !checked.Verified {
		return fmt.Errorf("Fairy admission failed: diagnostics=%+v unsupported=%+v", checked.Semantic.Diagnostics, checked.Semantic.Unsupported)
	}
	unit, err := cbackend.Emit(checked.MIR, "main::main", cbackend.Stage0WindowsMSVCProfile())
	if err != nil {
		return err
	}
	tc, err := cdriver.ResolveWindowsHostMSVC()
	if err != nil {
		return err
	}
	rt, err := cruntime.ScalarSource(128)
	if err != nil {
		return err
	}
	// Include the original C file; its ordinary relative legacy.h include is
	// resolved by the C compiler. Never edit or call generated private names.
	companion := "#include \"" + filepath.ToSlash(filepath.Join(root, "legacy.c")) + "\"\n" + rt
	artifact, err := cdriver.BuildWindowsMSVCABIProbe(tc, profile, unit.Source, companion)
	if err != nil {
		return err
	}
	if len(artifact.Objects) != 2 {
		return fmt.Errorf("expected separate Fairy and C objects, got %d", len(artifact.Objects))
	}
	result, err := cdriver.ExecuteWindowsMSVCABIProbe(artifact)
	if err != nil {
		return err
	}
	// The Windows C stdout text stream translates LF to CRLF.
	if result.ExitCode != 0 || result.Stdout != "accepted=2 callbacks=5\r\n" || result.Stderr != "" {
		return fmt.Errorf("interop failed: exit=%d stdout=%q stderr=%q", result.ExitCode, result.Stdout, result.Stderr)
	}
	fmt.Print(result.Stdout)
	fmt.Printf("interop ok: %s, MSVC %s, two separately compiled objects\n", profile.Name, tc.Version)
	return nil
}
