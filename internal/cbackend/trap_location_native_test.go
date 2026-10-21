package cbackend_test

import (
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/cbackend"
	"github.com/0disoft/fairyc/internal/cdriver"
	"github.com/0disoft/fairyc/internal/cruntime"
	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/target"
)

func TestEmittedTrapSourceLocationsNative(t *testing.T) {
	id, err := cdriver.HostNativeTarget()
	if err != nil {
		t.Skip(err)
	}
	toolchain, err := cdriver.ResolveNativeToolchain(id, "")
	if err != nil {
		t.Fatal(err)
	}
	profile := cbackend.Stage0Profile()
	renderRuntime := cruntime.LinuxRunSource
	if id == target.X8664WindowsMSVC {
		profile = cbackend.Stage0WindowsMSVCProfile()
		renderRuntime = cruntime.WindowsRunSource
	}
	support, err := renderRuntime(cdriver.NativeTrapExitBase)
	if err != nil {
		t.Fatal(err)
	}
	for _, sample := range []struct{ name, path, code, expression, category string }{
		{"second-bounds", "src/한\"글\\main.fy", `module main;
fn i32 main() {
    let ref u8[] data = b"a";
    let u8 first = data[0];
    let u8 second = data[1];
    return if (first == second) { 0 } else { 1 };
}`, "data[1]", "bounds"},
		{"constant-slice-callee", "main.fy", `module main;
fn i32 trapped() {
    let ref u8[] data = b"a";
    let ref u8[] part = data[0..2];
    return 0;
}
fn i32 main() { return trapped(); }
`, "data[0..2]", "bounds"},
		{"constant-direct-index-callee", "main.fy", `module main;
fn i32 trapped() { return b"a"[1] as i32; }
fn i32 main() { return trapped(); }
`, `b"a"[1]`, "bounds"},
		{"second-assert", "main.fy", `module main;
fn i32 main() {
    let u8 first = 1; assert(first == 1);
    let u8 second = 2; assert(second == 0);
    return 0;
}`, "assert(second == 0);", "assertion-failed"},
		{"dynamic-shift", "main.fy", `module main;
fn i32 shift(i32 value, i32 count) { return value << count; }
fn i32 main() { return shift(1, 32); }
`, "value << count", "invalid-shift"},
		{"constant-bounds-callee", "main.fy", `module main;
fn i32 trapped() {
    let ref u8[] data = b"a";
    return data[1] as i32;
}
fn i32 wrapper() { return trapped(); }
fn i32 main() {
    let i32 value = wrapper();
    assert(value == 0);
    return 0;
}`, "data[1]", "bounds"},
		{"qualified-dynamic-shift", "main.fy", `module main;
fn i32 shift(i32 value, i32 count) { return value << count; }
fn i32 main() { return shift(1, 32); }
`, "value << count", "invalid-shift"},
		{"unknown-long-source", strings.Repeat("s", 257) + ".fy", "module main; fn i32 main() { assert(false); return 0; }", "assert(false);", "assertion-failed"},
	} {
		t.Run(sample.name, func(t *testing.T) {
			program := lowerSource(t, sample.code)
			sampleProfile := profile
			qualified := sample.name == "qualified-dynamic-shift"
			sampleProfile.PackageTrapSources = qualified
			// Exercise source-display bytes independently of module-path grammar.
			for i := range program.Functions {
				for j := range program.Functions[i].Blocks {
					block := &program.Functions[i].Blocks[j]
					block.Terminator.Span.Source = sample.path
					if qualified {
						block.Terminator.Span.Package = hir.PackageID(2)
					}
					for k := range block.Operations {
						block.Operations[k].Span.Source = sample.path
						if qualified {
							block.Operations[k].Span.Package = hir.PackageID(2)
						}
					}
				}
			}
			unit, err := cbackend.Emit(program, "main::main", sampleProfile)
			if err != nil {
				t.Fatal(err)
			}
			for _, optimization := range []cdriver.Profile{cdriver.LowProfile, cdriver.OptimizedProfile} {
				artifact, err := toolchain.Build(cdriver.NativeBuildOptions{Profile: optimization, UnitSource: unit.Source, RuntimeSource: support, TrapControl: true})
				if err != nil {
					t.Fatalf("%v\n%s", err, unit.Source)
				}
				outcome, err := toolchain.Execute(artifact, cdriver.NativeExecutionOptions{RejectOutput: true})
				if err != nil || outcome.TrapCategory != sample.category {
					t.Fatalf("%s: %+v, %v", optimization.Name, outcome, err)
				}
				if len(sample.path) > 256 {
					if outcome.TrapLocation != nil {
						t.Fatal("overlong source acquired an inexact location")
					}
					continue
				}
				start := strings.Index(sample.code, sample.expression)
				want := cdriver.TrapLocation{Source: sample.path, StartByte: uint64(start), EndByte: uint64(start + len(sample.expression))}
				if qualified {
					want.Source = cbackend.PackageTrapSource(hir.PackageID(2), sample.path)
				}
				if outcome.TrapLocation == nil || *outcome.TrapLocation != want {
					t.Fatalf("%s: location %+v, want %+v", optimization.Name, outcome.TrapLocation, want)
				}
			}
		})
	}
}
