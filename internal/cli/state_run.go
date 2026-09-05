package cli

import (
	"fmt"
	"io"
	"runtime"

	"github.com/0disoft/fairyc/internal/cdriver"
	"github.com/0disoft/fairyc/internal/compiler"
	"github.com/0disoft/fairyc/internal/contractstatenative"
	"github.com/0disoft/fairyc/internal/cstdlib"
	"github.com/0disoft/fairyc/internal/frontend"
	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/packagegraph"
	"github.com/0disoft/fairyc/internal/semantic"
	"github.com/0disoft/fairyc/internal/source"
	"github.com/0disoft/fairyc/internal/target"
)

func runState(arguments []string, sourceRoot string, stdout, stderr io.Writer) int {
	if len(arguments) != 3 || arguments[1] != "--profile" || (arguments[2] != "low" && arguments[2] != "optimized") {
		return invocationError(stderr, "run-state expects FILE.fy --profile low|optimized")
	}
	if runtime.GOOS != "windows" || runtime.GOARCH != "amd64" {
		return buildFailure(stderr, fmt.Errorf("run-state requires Windows amd64 with x64 MSVC"))
	}
	limits := source.DefaultLimits()
	loaded, err := loadSource(sourceRoot, arguments[0], limits.MaxBytes)
	if err != nil {
		return inputError(stderr, err)
	}
	parsed, err := frontend.AnalyzeSyntax(loaded.file, limits)
	if err != nil {
		return internalError(stderr)
	}
	if len(parsed.Tree.Diagnostics) != 0 {
		if err := writeDiagnostics(stderr, loaded.file.RelativePath, parsed.Tree.Source, parsed.Tree.Diagnostics); err != nil {
			return outputError(stderr)
		}
		return ExitSourceDiagnostics
	}
	compiled, err := compiler.CheckExperimentalState(semantic.File{Package: hir.DefaultPackage, Path: loaded.file.RelativePath, Module: parsed.AST})
	if err != nil {
		return subsetCapabilityError(stderr, "experimental source state", err)
	}
	if !compiled.Verified {
		return subsetCapabilityError(stderr, "experimental source state", fmt.Errorf("state declaration or effects failed semantic validation"))
	}
	owner := compiled.StateTransactions[0]
	coordinate, err := packagegraph.ComputeCoordinateID(packagegraph.Coordinate{PackageUID: owner.Program.Layout.ContractUID, VersionIdentity: "experimental-state-source/1"})
	if err != nil {
		return internalError(stderr)
	}
	handoff, err := contractstatenative.PrepareCompiled(compiled, owner.Package, owner.Function, coordinate)
	if err != nil {
		return subsetCapabilityError(stderr, "native state", err)
	}
	unit, err := contractstatenative.Emit(handoff, target.X8664WindowsMSVC)
	if err != nil {
		return subsetCapabilityError(stderr, "native state", err)
	}
	support, err := cstdlib.ContractStateRuntimeSource()
	if err != nil {
		return internalError(stderr)
	}
	toolchain, err := cdriver.ResolveWindowsHostMSVC()
	if err != nil {
		return buildFailure(stderr, err)
	}
	profile := cdriver.LowProfile
	if arguments[2] == "optimized" {
		profile = cdriver.OptimizedProfile
	}
	artifact, err := cdriver.BuildWindowsHost(toolchain, profile, unit.Source+stateRunHost, support)
	if err != nil {
		return buildFailure(stderr, err)
	}
	outcome, err := cdriver.ExecuteWindowsHost(artifact)
	if err != nil {
		return buildFailure(stderr, err)
	}
	if outcome.ExitCode != 0 || outcome.Stderr != "" {
		return buildFailure(stderr, fmt.Errorf("native state host failed with status %d", outcome.ExitCode))
	}
	if _, err := io.WriteString(stdout, outcome.Stdout); err != nil {
		return outputError(stderr)
	}
	return ExitSuccess
}

// The command starts a fresh in-memory store on every invocation. The receipt reports native
// transaction outcome independently of the command's successful execution status.
const stateRunHost = `
#include <stdio.h>
int main(void) {
    fycs_native_store_v0 store;
    fycs_native_store_initialize_v0(&store, fycs_native_layout_hash_v0, fycs_native_body_hash_v0);
    int32_t status = fycs_native_entry_v0(&store);
    if (status < 0 || status > 2 || fycs_native_live_resources_v0() != 0u) { return 1; }
    printf("state exit: %s\n", status == 0 ? "commit" : status == 1 ? "reject" : "trap");
    if (status == 2) { printf("trap category: %s\n", (const char *)fycs_native_trap_category_v0); }
    printf("state value: ");
    if (store.present == 0u) { printf("absent"); }
    for (size_t i = 0; i < store.length; ++i) { printf("%02x", (unsigned int)store.value[i]); }
    printf("\n");
    return ferror(stdout) ? 1 : 0;
}
`
