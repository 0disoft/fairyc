package cli

import (
	"bytes"
	"reflect"
	"runtime"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/atlasmap"
	"github.com/0disoft/fairyc/internal/cdriver"
	"github.com/0disoft/fairyc/internal/nativesupport"
	"github.com/0disoft/fairyc/internal/target"
)

const implicitReceiverReturnedSource = `module main;
resource Token{i32 value;cleanup{}}
fn ref Token Token::view(ref Token this){return this;}
fn void Token::set(ref mut Token this,i32 n){this.value=n;}
fn i32 main(){var token=Token{.value=1};let view=token.view();assert(view.value==1);token.set(42);assert(token.value==42);return 0;}`

func implicitReceiverExecutionSources() []struct{ name, source string } {
	return []struct{ name, source string }{
		{"shared", `module main;resource Token{i32 value;cleanup{assert(this.value==42);}}fn i32 Token::read(ref Token this){return this.value;}fn i32 main(){let token=Token{.value=42};assert(token.read()==42);assert(token.read()==42 && token.value==42);return 0;}`},
		{"exclusive", `module main;resource Token{i32 value;cleanup{assert(this.value==42);}}fn void Token::set(ref mut Token this,i32 n){this.value=n;}fn i32 main(){var token=Token{.value=1};token.set(42);assert(token.value==42);return 0;}`},
		{"returned", implicitReceiverReturnedSource},
		{"mutable-returned", `module main;resource Token{i32 value;cleanup{assert(this.value==42);}}fn ref mut Token Token::edit(ref mut Token this){return this;}fn i32 main(){var token=Token{.value=1};let view=token.edit();view.value=42;assert(token.value==42);return 0;}`},
		{"generic", `module main;struct Cell{i32 value;}fn i32 Cell::get(ref Cell this){return this.value;}fn void Cell::set(ref mut Cell this,i32 n){this.value=n;}fn i32 exercise<T>(T marker){discard marker;var cell=Cell{.value=1};cell.set(42);assert(cell.get()==42);return 0;}fn i32 main(){assert(exercise<i32>(1)==0 && exercise<u8>(2)==0);return 0;}`},
		{"cleanup", `module main;struct Cell{i32 value;}fn void Cell::set(ref mut Cell this,i32 n){this.value=n;}resource Token{i32 value;cleanup{var cell=Cell{.value=1};cell.set(this.value);assert(cell.value==42);}}fn i32 main(){let token=Token{.value=42};return 0;}`},
	}
}

func atlasImplicitReceiverBorrowCases() []atlasBorrowOracleFixture {
	var cases []atlasBorrowOracleFixture
	for _, f := range atlasMoveImplicitBorrowCases() {
		cases = append(cases, atlasBorrowOracleFixture{name: "implicit-" + f.name, source: f.source})
	}
	// Atlas exposes ordinary functions and concrete generic cleanup owners.
	// Nongeneric private drop glue remains outside the Go snapshot contract;
	// its real execution is covered by implicitReceiverExecutionSources.
	return append(cases,
		atlasBorrowOracleFixture{name: "implicit-returned", source: implicitReceiverReturnedSource},
		atlasBorrowOracleFixture{name: "implicit-mutable-returned", source: `module main;struct Cell{i32 value;}fn ref mut Cell Cell::edit(ref mut Cell this){return this;}fn i32 main(){var cell=Cell{.value=1};let view=cell.edit();view.value=42;return cell.value;}`},
		atlasBorrowOracleFixture{name: "implicit-generic", source: `module main;struct Cell{i32 value;}fn ref Cell Cell::view(ref Cell this){return this;}fn i32 exercise<T>(T marker){discard marker;let cell=Cell{.value=42};let view=cell.view();return view.value;}fn i32 main(){return exercise<i32>(1)+exercise<u8>(2);}`},
		atlasBorrowOracleFixture{name: "implicit-cleanup", source: `module main;struct Cell{i32 value;}fn ref Cell Cell::view(ref Cell this){return this;}resource Token<T>{T value;cleanup{let cell=Cell{.value=42};let view=cell.view();assert(view.value==42);}}fn i32 main(){let a=Token<i32>{.value=1};let b=Token<u8>{.value=2};return 0;}`},
	)
}

func TestCandidateImplicitReceiverExecution(t *testing.T) {
	if runtime.GOOS != "windows" || runtime.GOARCH != "amd64" {
		t.Skip("Windows implicit receiver replay; Linux execution is batched")
	}
	fixtures := implicitReceiverExecutionSources()
	var roots []string
	for _, f := range fixtures {
		root := writeWorkspaceSessionFixture(t, f.source)
		var diagnostic bytes.Buffer
		if _, code := prepareAtlasWorkspaceSchema(root, false, &diagnostic, atlasmap.LogicalSchemaV3); code != ExitSuccess {
			t.Fatalf("Go implicit receiver %s: %d %s", f.name, code, &diagnostic)
		}
		roots = append(roots, root)
	}
	toolchain, err := cdriver.ResolveWindowsHostMSVC()
	if err != nil {
		t.Fatal(err)
	}
	support, err := nativesupport.Source(target.X8664WindowsMSVC, nil, nativesupport.Options{})
	if err != nil {
		t.Fatal(err)
	}
	candidateAtlasProbeSize(t, "project_atlas_source", candidateRegionBytesMain(), 2<<20, func(t *testing.T, image string) {
		t.Setenv("PATH", "")
		for i, root := range roots {
			t.Run(fixtures[i].name, func(t *testing.T) {
				before := remoteTree(t, root)
				code, out, stderr := fetchCLIProcess(t, image, root, nil, "")
				if code != 0 || stderr != "" || !strings.Contains(out, "fy_candidate_entry") {
					t.Fatalf("implicit receiver interpreter/emission: %d %.2000s %q", code, out, stderr)
				}
				for _, profile := range []cdriver.Profile{cdriver.LowProfile, cdriver.OptimizedProfile} {
					native, err := cdriver.BuildWindowsMSVCTargetParts(toolchain, profile, "/* single-buffer transport */\n", "typedef int fy_empty_literals;\n", []string{out}, support)
					if err != nil {
						t.Fatal(err)
					}
					candidateAggregateExecuteImage(t, native.Bytes, 0, "")
				}
				if !reflect.DeepEqual(before, remoteTree(t, root)) {
					t.Fatal("implicit receiver execution changed source bytes")
				}
			})
		}
	}, "project_compiler_core", "module_unit_c_emit", "module_unit_interpreter")
}
