package cbackend_test

import (
	"errors"
	"fmt"
	"slices"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/cbackend"
	"github.com/0disoft/fairyc/internal/frontend"
	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/interpreter"
	"github.com/0disoft/fairyc/internal/mir"
	"github.com/0disoft/fairyc/internal/semantic"
	"github.com/0disoft/fairyc/internal/source"
	"github.com/0disoft/fairyc/internal/target"
)

func TestEmitScalarControlUnit(t *testing.T) {
	t.Parallel()
	program := lowerSource(t, "module main;\nfn add(left: i32, right: i32) -> i32 { left + right }\nfn main() -> i32 { var count: i32 = 0; while count < 3 { count = count + 1; } add(count, 39) }\n")
	outcome, err := interpreter.Execute(program, "main::main")
	if err != nil || outcome.Kind != interpreter.OutcomeReturn || outcome.Value != 42 {
		t.Fatalf("interpreter = %+v, %v", outcome, err)
	}
	unit, err := cbackend.Emit(program, "main::main", cbackend.Stage0Profile())
	if err != nil {
		t.Fatalf("Emit() error = %v", err)
	}
	if unit.Target != string(target.X8664LinuxGNU) || unit.Dialect != cbackend.DialectC11 || unit.Entry != "main::main" {
		t.Fatalf("Emit() identity = %+v", unit)
	}
	if !slices.Equal(unit.Functions, []string{"main::add", "main::main"}) {
		t.Fatalf("Emit() functions = %v", unit.Functions)
	}
	wantFeatures := []string{"checked-arithmetic", "scalar-call", "scalar-comparison", "scalar-control-flow", "trap"}
	if !slices.Equal(unit.Features, wantFeatures) {
		t.Fatalf("Emit() features = %v, want %v", unit.Features, wantFeatures)
	}
	for _, required := range []string{
		"#include <stdint.h>",
		"#define FY_TRAP_INTEGER_OVERFLOW UINT32_C(2)",
		"_Noreturn void fy_runtime_trap(uint32_t category);",
		"static int32_t fy_fn_0(int32_t",
		"int main(void)",
	} {
		if !strings.Contains(unit.Source, required) {
			t.Errorf("generated C is missing %q", required)
		}
	}
	if strings.Contains(unit.Source, "abort(") || strings.Contains(unit.Source, "#include <stdlib.h>") {
		t.Fatalf("generated C owns trap termination instead of delegating to the runtime:\n%s", unit.Source)
	}
	second, err := cbackend.Emit(program, "main::main", cbackend.Stage0Profile())
	if err != nil || second.Source != unit.Source {
		t.Fatalf("second Emit() differs: error=%v", err)
	}
}

func TestEmitNonEmptyResourceDropBodyAsVerifiedGlue(t *testing.T) {
	t.Parallel()
	program := lowerSource(t, `module main;
fn observe(value: i32) -> i32 { value }
resource Token {
    value: i32;
    drop fn(self: &mut Self) {
        let observed: i32 = observe(self.value);
    }
}
fn main() -> i32 {
    let token: Token = Token { value: 65 };
    42
}
`)
	unit, err := cbackend.Emit(program, "main::main", cbackend.Stage0WindowsMSVCProfile())
	if err != nil {
		t.Fatalf("Emit() error = %v", err)
	}
	dropIndex := slices.Index(unit.Functions, "main::$drop$Token")
	observeIndex := slices.Index(unit.Functions, "main::observe")
	if dropIndex < 0 || observeIndex < 0 || !slices.Contains(unit.Features, "non-empty-resource-drop") {
		t.Fatalf("Emit() identity = functions %v, features %v", unit.Functions, unit.Features)
	}
	if !strings.Contains(unit.Source, fmt.Sprintf("fy_fn_%d(fy_l_", dropIndex)) ||
		!strings.Contains(unit.Source, fmt.Sprintf("fy_fn_%d(", observeIndex)) {
		t.Fatalf("generated C does not call verified drop glue and its typed body:\n%s", unit.Source)
	}
}

func TestEmitWindowsMSVCProfileUsesLLP64TargetFacts(t *testing.T) {
	t.Parallel()
	program := lowerSource(t, `module main;
@link_name("fairyc_unused_long")
unsafe extern C fn unused_long(value: c::long) -> c::long;
fn main() -> i32 { 42 }
`)
	unit, err := cbackend.Emit(program, "main::main", cbackend.Stage0WindowsMSVCProfile())
	if err != nil {
		t.Fatalf("Emit(Windows MSVC) error = %v", err)
	}
	if unit.Target != string(target.X8664WindowsMSVC) || unit.Dialect != cbackend.DialectC11 || unit.Instrumentation != "" {
		t.Fatalf("Emit(Windows MSVC) identity = %+v", unit)
	}
	for _, required := range []string{
		"/* fairyc constrained C transport: x86_64-windows-msvc, ISO C11 */",
		"_Static_assert(sizeof(long) == UINT64_C(4), \"fairyc target c.long size\");",
		"_Static_assert(_Alignof(long) == UINT64_C(4), \"fairyc target c.long alignment\");",
		"extern long fairyc_unused_long(long fy_a_0);",
	} {
		if !strings.Contains(unit.Source, required) {
			t.Errorf("Windows MSVC generated C is missing %q:\n%s", required, unit.Source)
		}
	}
	if strings.Contains(unit.Source, "_POSIX_C_SOURCE") || strings.Contains(unit.Source, "sys/types.h") {
		t.Fatalf("Windows MSVC generated C leaked POSIX providers:\n%s", unit.Source)
	}
}

func TestEmitWindowsMSVCProfileRejectsUnavailablePOSIXScalar(t *testing.T) {
	t.Parallel()
	program := lowerSource(t, `module main;
@link_name("fairyc_unused_ssize")
unsafe extern C fn unused_ssize(value: c::ssize_t) -> c::ssize_t;
fn main() -> i32 { 42 }
`)
	_, err := cbackend.Emit(program, "main::main", cbackend.Stage0WindowsMSVCProfile())
	var capability cbackend.CapabilityError
	if !errors.As(err, &capability) || !strings.Contains(capability.Feature, `C scalar "c.ssize_t" is unavailable for target "x86_64-windows-msvc"`) {
		t.Fatalf("Emit(Windows MSVC c.ssize_t) error = %v", err)
	}
}

func TestEmitScalarForeignCall(t *testing.T) {
	t.Parallel()
	program := lowerSource(t, "module main;\n@link_name(\"fairyc_fixture_add_one\")\nunsafe extern C fn add_one(value: i32) -> i32;\nfn main() -> i32 { unsafe { add_one(41) } }\n")
	_, interpreterErr := interpreter.Execute(program, "main::main")
	var interpreterCapability interpreter.CapabilityError
	if !errors.As(interpreterErr, &interpreterCapability) || interpreterCapability.Kind != "foreign-call" {
		t.Fatalf("interpreter error = %v", interpreterErr)
	}
	unit, err := cbackend.Emit(program, "main::main", cbackend.Stage0Profile())
	if err != nil {
		t.Fatalf("Emit() error = %v", err)
	}
	if !slices.Equal(unit.Features, []string{"foreign-c-call", "scalar-call"}) {
		t.Fatalf("Emit() features = %v", unit.Features)
	}
	for _, required := range []string{
		"extern int32_t fairyc_fixture_add_one(int32_t fy_a_0);",
		"fairyc_fixture_add_one(INT32_C(41));",
	} {
		if !strings.Contains(unit.Source, required) {
			t.Errorf("generated C is missing %q:\n%s", required, unit.Source)
		}
	}
	if strings.Contains(unit.Source, "static int32_t fairyc_fixture_add_one") {
		t.Fatalf("generated C turned an imported symbol into an internal definition:\n%s", unit.Source)
	}
	if strings.Contains(unit.Source, "extern int32_t add_one(") {
		t.Fatalf("generated C used the Fairy declaration name instead of @link_name:\n%s", unit.Source)
	}
	if strings.Contains(unit.Source, "int32_t fy_l_0;") {
		t.Fatalf("generated C declared a phantom local for the foreign symbol:\n%s", unit.Source)
	}
}

func TestEmitTracksOnlyCalledForeignSymbolsInSortedOrder(t *testing.T) {
	t.Parallel()
	program := lowerSource(t, `module main;
@link_name("fairyc_fixture_z")
unsafe extern C fn call_z(value: i32) -> i32;
@link_name("fairyc_fixture_a")
unsafe extern C fn call_a(value: i32) -> i32;
@link_name("fairyc_fixture_unused")
unsafe extern C fn unused(value: i32) -> i32;
fn main() -> i32 {
    let first: i32 = unsafe { call_z(20) };
    let second: i32 = unsafe { call_a(first) };
    unsafe { call_a(second) }
}
`)
	unit, err := cbackend.Emit(program, "main::main", cbackend.Stage0Profile())
	if err != nil {
		t.Fatalf("Emit() error = %v", err)
	}
	want := []string{"fairyc_fixture_a", "fairyc_fixture_z"}
	if !slices.Equal(unit.ForeignSymbols, want) {
		t.Fatalf("Emit() foreign symbols = %v, want %v", unit.ForeignSymbols, want)
	}
}

func TestEmitDeduplicatesCompilerOwnedCStringStorage(t *testing.T) {
	t.Parallel()
	program := lowerSource(t, `module main;
fn main() -> i32 {
    let first: &CStr = c"fairy";
    let second: &CStr = c"fairy";
    42
}
`)
	unit, err := cbackend.Emit(program, "main::main", cbackend.Stage0Profile())
	if err != nil {
		t.Fatalf("Emit() error = %v", err)
	}
	if !slices.Equal(unit.Features, []string{"static-c-string"}) {
		t.Fatalf("features = %v", unit.Features)
	}
	if strings.Count(unit.Source, "static const uint8_t fy_static_cstr_") != 1 {
		t.Fatalf("generated C did not deduplicate C string storage:\n%s", unit.Source)
	}
	for _, required := range []string{
		"[6] = { UINT8_C(102), UINT8_C(97), UINT8_C(105), UINT8_C(114), UINT8_C(121), UINT8_C(0) };",
		"const uint8_t * fy_l_",
		" = fy_static_cstr_",
	} {
		if !strings.Contains(unit.Source, required) {
			t.Errorf("generated C is missing %q:\n%s", required, unit.Source)
		}
	}
	second, err := cbackend.Emit(program, "main::main", cbackend.Stage0Profile())
	if err != nil || second.Source != unit.Source {
		t.Fatalf("second Emit() differs: error=%v", err)
	}
}

func TestEmitCStringLiteralDataAddressForForeignCall(t *testing.T) {
	t.Parallel()
	program := lowerSource(t, `module main;
@link_name("fairyc_fixture_is_fairy")
unsafe extern C fn is_fairy(value: *u8) -> u32;
fn main() -> i32 {
    if unsafe { is_fairy(c"fairy" as *u8) } == 1 { 42 } else { 1 }
}
`)
	unit, err := cbackend.Emit(program, "main::main", cbackend.Stage0Profile())
	if err != nil {
		t.Fatalf("Emit() error = %v", err)
	}
	for _, feature := range []string{"cstr-data-address", "foreign-c-call", "static-c-string"} {
		if !slices.Contains(unit.Features, feature) {
			t.Errorf("features %v are missing %q", unit.Features, feature)
		}
	}
	if !slices.Equal(unit.ForeignSymbols, []string{"fairyc_fixture_is_fairy"}) {
		t.Fatalf("foreign symbols = %v", unit.ForeignSymbols)
	}
	for _, required := range []string{
		"extern uint32_t fairyc_fixture_is_fairy(const uint8_t * fy_a_0);",
		"static const uint8_t fy_static_cstr_",
		" = fy_static_cstr_",
		"fairyc_fixture_is_fairy(fy_l_",
	} {
		if !strings.Contains(unit.Source, required) {
			t.Errorf("generated C is missing %q:\n%s", required, unit.Source)
		}
	}
}

func TestEmitDeduplicatesCompilerOwnedByteStringStorage(t *testing.T) {
	t.Parallel()
	program := lowerSource(t, `module main;
fn main() -> i32 {
    let first: &[u8] = b"fairy";
    let second: &[u8] = b"fairy";
    let empty: &[u8] = b"";
    42
}
`)
	unit, err := cbackend.Emit(program, "main::main", cbackend.Stage0Profile())
	if err != nil {
		t.Fatalf("Emit() error = %v", err)
	}
	if !slices.Equal(unit.Features, []string{"static-byte-string"}) {
		t.Fatalf("features = %v", unit.Features)
	}
	if strings.Count(unit.Source, "static const uint8_t fy_static_bytes_") != 2 {
		t.Fatalf("generated C did not deduplicate byte-string storage or preserve the empty sentinel:\n%s", unit.Source)
	}
	for _, required := range []string{
		"typedef struct fy_byte_slice",
		"const uint8_t *data;",
		"uintptr_t length;",
		"[5] = { UINT8_C(102), UINT8_C(97), UINT8_C(105), UINT8_C(114), UINT8_C(121) };",
		"[1] = { UINT8_C(0) };",
		"fy_byte_slice fy_l_",
		"(void)&fy_l_",
		"((uintptr_t)UINT64_C(5))",
		"((uintptr_t)UINT64_C(0))",
	} {
		if !strings.Contains(unit.Source, required) {
			t.Errorf("generated C is missing %q:\n%s", required, unit.Source)
		}
	}
	second, err := cbackend.Emit(program, "main::main", cbackend.Stage0Profile())
	if err != nil || second.Source != unit.Source {
		t.Fatalf("second Emit() differs: error=%v", err)
	}
}

func TestEmitCheckedImmutableByteViewIndex(t *testing.T) {
	t.Parallel()
	program := lowerSource(t, `module main;
fn main() -> i32 {
    let value: u8 = b"\x00A"[1];
    if value == 65 { 42 } else { 1 }
}
`)
	unit, err := cbackend.Emit(program, "main::main", cbackend.Stage0Profile())
	if err != nil {
		t.Fatalf("Emit() error = %v", err)
	}
	for _, feature := range []string{"static-byte-string", "trap", "view-index"} {
		if !slices.Contains(unit.Features, feature) {
			t.Errorf("features %v are missing %q", unit.Features, feature)
		}
	}
	for _, required := range []string{
		"FY_TRAP_BOUNDS",
		".length) { fy_pending_trap = FY_TRAP_BOUNDS;",
		".data[",
	} {
		if !strings.Contains(unit.Source, required) {
			t.Errorf("generated C is missing %q:\n%s", required, unit.Source)
		}
	}
}

func TestEmitCheckedScalarFixedArrayIndex(t *testing.T) {
	t.Parallel()
	program := lowerSource(t, `module main;
fn pick(values: [u8; 3], index: usize) -> u8 { values[index] }
fn view_pick(values: &[u8], index: usize) -> u8 { values[index] }
fn main() -> i32 {
    let values: [u8; 3] = [40, 41, 42];
    if values.len == 3 && view_pick(&values, 0) == 40 { pick(values, 2) as i32 } else { 1 }
}

`)
	unit, err := cbackend.Emit(program, "main::main", cbackend.Stage0Profile())
	if err != nil {
		t.Fatalf("Emit() error = %v", err)
	}
	for _, feature := range []string{"array-byte-view-borrow", "array-index", "array-length", "scalar-fixed-array", "trap", "view-index"} {
		if !slices.Contains(unit.Features, feature) {
			t.Errorf("features %v are missing %q", unit.Features, feature)
		}
	}
	for _, required := range []string{
		"typedef struct { uint8_t data[UINT64_C(3)]; } fy_array_0;",
		"(fy_array_0){ .data = { UINT8_C(40), UINT8_C(41), UINT8_C(42) } }",
		"(fy_byte_slice){ .data = fy_l_",
		"= ((uintptr_t)UINT64_C(3));",
		">= ((uintptr_t)UINT64_C(3))",
		".data[",
	} {
		if !strings.Contains(unit.Source, required) {
			t.Errorf("generated C is missing %q:\n%s", required, unit.Source)
		}
	}
	second, err := cbackend.Emit(program, "main::main", cbackend.Stage0Profile())
	if err != nil || second.Source != unit.Source {
		t.Fatalf("second Emit() differs: error=%v", err)
	}
}

func TestEmitCheckedMutableIndexedStores(t *testing.T) {
	t.Parallel()
	program := lowerSource(t, `module main;
fn update(bytes: &mut [u8]) -> u8 {
    bytes[1] = 42;
    bytes[1]
}
fn main() -> i32 {
    var values: [u8; 3] = [1, 2, 3];
    values[0] = 40;
    update(&mut values) as i32
}
`)
	unit, err := cbackend.Emit(program, "main::main", cbackend.Stage0Profile())
	if err != nil {
		t.Fatalf("Emit() error = %v", err)
	}
	for _, feature := range []string{"array-index-store", "mutable-view-index-store", "view-index"} {
		if !slices.Contains(unit.Features, feature) {
			t.Errorf("features %v are missing %q", unit.Features, feature)
		}
	}
	for _, required := range []string{
		"] = UINT8_C(40);",
		"] = UINT8_C(42);",
		"FY_TRAP_BOUNDS",
	} {
		if !strings.Contains(unit.Source, required) {
			t.Errorf("generated C is missing %q:\n%s", required, unit.Source)
		}
	}
}

func TestEmitReassignedFixedArrayByteView(t *testing.T) {
	t.Parallel()
	program := lowerSource(t, `module main;
fn main() -> i32 {
    let first: [u8; 3] = [1, 2, 3];
    let second: [u8; 3] = [40, 41, 42];
    var bytes: &[u8] = &first;
    bytes = &second;
    bytes[2] as i32
}
`)
	unit, err := cbackend.Emit(program, "main::main", cbackend.Stage0Profile())
	if err != nil {
		t.Fatalf("Emit() error = %v", err)
	}
	for _, feature := range []string{"array-byte-view-borrow", "scalar-fixed-array", "view-index"} {
		if !slices.Contains(unit.Features, feature) {
			t.Errorf("features %v are missing %q", unit.Features, feature)
		}
	}
	if strings.Count(unit.Source, "(fy_byte_slice){ .data = fy_l_") != 2 || !strings.Contains(unit.Source, ".data[") {
		t.Fatalf("generated C does not preserve both view definitions and the selected read:\n%s", unit.Source)
	}
}

func TestEmitMutableFixedArrayByteViewDataAddress(t *testing.T) {
	t.Parallel()
	program := lowerSource(t, `module main;
unsafe extern C fn fill(data: *mut u8, length: usize) -> u32;
fn fill_safe(bytes: &mut [u8]) -> u32 {
    unsafe { fill(bytes as *mut u8, bytes.len) }
}
fn main() -> i32 {
    var values: [u8; 3] = [0, 0, 0];
    if fill_safe(&mut values) == 0 { 42 } else { 1 }
}
`)
	unit, err := cbackend.Emit(program, "main::main", cbackend.Stage0Profile())
	if err != nil {
		t.Fatalf("Emit() error = %v", err)
	}
	for _, feature := range []string{"byte-slice-data-address", "mutable-array-byte-view-borrow", "scalar-fixed-array"} {
		if !slices.Contains(unit.Features, feature) {
			t.Errorf("features %v are missing %q", unit.Features, feature)
		}
	}
	for _, required := range []string{
		"typedef struct fy_mut_byte_slice {", "uint8_t *data;", "fy_mut_byte_slice fy_l_",
		"(fy_mut_byte_slice){ .data = fy_l_", ".data;", "fill(fy_l_",
	} {
		if !strings.Contains(unit.Source, required) {
			t.Errorf("generated C is missing %q:\n%s", required, unit.Source)
		}
	}
}

func TestEmitArrayByteViewDeclaresTransportWithoutLiteralStorage(t *testing.T) {
	t.Parallel()
	program := lowerSource(t, `module main;
fn length(bytes: &[u8]) -> usize { bytes.len }
fn main() -> i32 {
    let values: [u8; 3] = [40, 41, 42];
    if length(&values) == 3 { 42 } else { 1 }
}
`)
	unit, err := cbackend.Emit(program, "main::main", cbackend.Stage0Profile())
	if err != nil {
		t.Fatalf("Emit() error = %v", err)
	}
	if !strings.Contains(unit.Source, "typedef struct fy_byte_slice {") || strings.Contains(unit.Source, "static const uint8_t fy_static_bytes_") {
		t.Fatalf("generated C does not declare the array-only immutable view transport:\n%s", unit.Source)
	}
}

func TestEmitRejectsFixedArrayBeyondExecutableElementLimit(t *testing.T) {
	t.Parallel()
	program := lowerSource(t, "module main;\nfn main() -> i32 { 42 }\n")
	program.Functions[0].ParameterTypes = []hir.Type{
		hir.FixedArray(hir.TypeU8, hir.MaxExecutableFixedArrayElements+1),
	}
	_, err := cbackend.Emit(program, "main::main", cbackend.Stage0Profile())
	if err == nil || !strings.Contains(err.Error(), "outside the bounded executable C subset") {
		t.Fatalf("Emit() error = %v", err)
	}
}

func TestEmitCheckedImmutableByteViewSlice(t *testing.T) {
	t.Parallel()
	program := lowerSource(t, `module main;
fn main() -> i32 {
    let bytes: &[u8] = b"fairy"[1..4];
    if bytes.len == 3 && bytes[0] == 97 && bytes[2] == 114 { 42 } else { 1 }
}
`)
	unit, err := cbackend.Emit(program, "main::main", cbackend.Stage0Profile())
	if err != nil {
		t.Fatalf("Emit() error = %v", err)
	}
	for _, feature := range []string{"static-byte-string", "trap", "view-slice", "view-index"} {
		if !slices.Contains(unit.Features, feature) {
			t.Errorf("features %v are missing %q", unit.Features, feature)
		}
	}
	for _, required := range []string{
		"FY_TRAP_BOUNDS",
		" || ",
		" > (",
		").length) { fy_pending_trap = FY_TRAP_BOUNDS;",
		"(fy_byte_slice){ .data = ",
		".data + (",
		".length = (",
	} {
		if !strings.Contains(unit.Source, required) {
			t.Errorf("generated C is missing %q:\n%s", required, unit.Source)
		}
	}
}

func TestEmitImmutableByteViewSliceWithOmittedBounds(t *testing.T) {
	t.Parallel()
	program := lowerSource(t, `module main;
fn main() -> i32 {
    let bytes: &[u8] = b"fairy"[1..];
    if bytes.len == 4 && bytes[3] == 121 { 42 } else { 1 }
}
`)
	unit, err := cbackend.Emit(program, "main::main", cbackend.Stage0Profile())
	if err != nil {
		t.Fatalf("Emit() error = %v", err)
	}
	for _, feature := range []string{"view-length", "view-slice", "view-index"} {
		if !slices.Contains(unit.Features, feature) {
			t.Errorf("features %v are missing %q", unit.Features, feature)
		}
	}
	if !strings.Contains(unit.Source, ".length;") || !strings.Contains(unit.Source, ".data + (") {
		t.Fatalf("generated C does not normalize the omitted end through retained length:\n%s", unit.Source)
	}
}

func TestEmitValuePreservingIntegerCast(t *testing.T) {
	t.Parallel()
	program := lowerSource(t, `module main;
fn main() -> i32 { b"*"[0] as i32 }
`)
	unit, err := cbackend.Emit(program, "main::main", cbackend.Stage0Profile())
	if err != nil {
		t.Fatalf("Emit() error = %v", err)
	}
	for _, feature := range []string{"scalar-cast", "static-byte-string", "trap", "view-index"} {
		if !slices.Contains(unit.Features, feature) {
			t.Errorf("features %v are missing %q", unit.Features, feature)
		}
	}
	if !strings.Contains(unit.Source, "= (int32_t)(fy_l_") {
		t.Fatalf("generated C is missing the value-preserving uint8_t-to-int32_t cast:\n%s", unit.Source)
	}
}

func TestEmitTypedUnsignedOrderedComparisons(t *testing.T) {
	t.Parallel()
	program := lowerSource(t, `module main;
fn main() -> i32 {
    let byte: u8 = b"*"[0];
    let word: u32 = byte as u32;
    let size: usize = word as usize;
    let pointer: *u8 = null;
    if byte > 41 && word >= 42 && size <= 42 && pointer == null { 42 } else { 1 }
}
`)
	unit, err := cbackend.Emit(program, "main::main", cbackend.Stage0Profile())
	if err != nil {
		t.Fatalf("Emit() error = %v", err)
	}
	for _, feature := range []string{"scalar-cast", "scalar-comparison", "scalar-control-flow", "view-index"} {
		if !slices.Contains(unit.Features, feature) {
			t.Errorf("features %v are missing %q", unit.Features, feature)
		}
	}
	for _, required := range []string{") > (UINT8_C(41))", ") >= (UINT32_C(42))", ") <= (((uintptr_t)UINT64_C(42)))", ") == (((void *)0))"} {
		if !strings.Contains(unit.Source, required) {
			t.Errorf("generated C is missing typed comparison %q:\n%s", required, unit.Source)
		}
	}
}

func TestEmitCheckedUnsignedArithmetic(t *testing.T) {
	t.Parallel()
	program := lowerSource(t, `module main;
fn main() -> i32 {
    let byte: u8 = 6;
    let byte_product: u8 = byte * 7;
    let word: u32 = 50;
    let word_difference: u32 = word - 8;
    let size: usize = 41;
    let size_sum: usize = size + 1;
    if byte_product == 42 && word_difference == 42 && size_sum == 42 { 42 } else { 1 }
}

`)
	unit, err := cbackend.Emit(program, "main::main", cbackend.Stage0Profile())
	if err != nil {
		t.Fatalf("Emit() error = %v", err)
	}
	for _, feature := range []string{"checked-arithmetic", "scalar-comparison", "scalar-control-flow", "trap"} {
		if !slices.Contains(unit.Features, feature) {
			t.Errorf("features %v are missing %q", unit.Features, feature)
		}
	}
	for _, required := range []string{
		"UINT8_MAX / (UINT8_C(7))",
		"if ((fy_l_",
		") < (UINT32_C(8))",
		"UINTPTR_MAX - (((uintptr_t)UINT64_C(1)))",
	} {
		if !strings.Contains(unit.Source, required) {
			t.Errorf("generated C is missing checked unsigned arithmetic %q:\n%s", required, unit.Source)
		}
	}
}

func TestEmitCheckedUnsignedDivision(t *testing.T) {
	t.Parallel()
	program := lowerSource(t, `module main;
fn main() -> i32 {
    let byte_input: u8 = 84;
    let byte: u8 = byte_input / 2;
    let word_input: u32 = 85;
    let word: u32 = word_input % 43;
    let size_input: usize = 84;
    let size: usize = size_input / 2;
    if byte == 42 && word == 42 && size == 42 { 42 } else { 1 }
}
`)
	unit, err := cbackend.Emit(program, "main::main", cbackend.Stage0Profile())
	if err != nil {
		t.Fatalf("Emit() error = %v", err)
	}
	for _, required := range []string{
		"volatile uint8_t fy_checked_divisor_u8;",
		"volatile uint32_t fy_checked_divisor_u32;",
		"volatile uintptr_t fy_checked_divisor_usize;",
		"/ fy_checked_divisor_u8;",
		"% fy_checked_divisor_u32;",
		"/ fy_checked_divisor_usize;",
	} {
		if !strings.Contains(unit.Source, required) {
			t.Errorf("generated C is missing checked unsigned division %q:\n%s", required, unit.Source)
		}
	}
	if strings.Contains(unit.Source, "FY_TRAP_SIGNED_DIVISION_OVERFLOW") {
		t.Fatalf("unsigned-only C unit includes signed division overflow machinery:\n%s", unit.Source)
	}
}

func TestEmitUnsignedBitwiseAndCheckedRightShift(t *testing.T) {
	t.Parallel()
	program := lowerSource(t, `module main;
fn main() -> i32 {
    let byte_input: u8 = 84;
    let byte: u8 = byte_input >> 1;
    let byte_not_input: u8 = 213;
    let byte_not: u8 = ~byte_not_input;
    let word_input: u32 = 43;
    let word: u32 = word_input & 63;
    let word_not_input: u32 = 4294967253;
    let word_not: u32 = ~word_not_input;
    let word_shift_input: u32 = 84;
    let word_shift: u32 = word_shift_input >> 1;
    let size_input: usize = 43;
    let size: usize = (size_input ^ 1) | 0;
    let size_not_input: usize = 18446744073709551573;
    let size_not: usize = ~size_not_input;
    let size_shift_input: usize = 84;
    let size_shift: usize = size_shift_input >> 1;
    if byte == 42 && byte_not == 42 && word == 43 && word_not == 42 && word_shift == 42 && size == 42 && size_not == 42 && size_shift == 42 { 42 } else { 1 }
}
`)
	unit, err := cbackend.Emit(program, "main::main", cbackend.Stage0Profile())
	if err != nil {
		t.Fatalf("Emit() error = %v", err)
	}
	for _, feature := range []string{"checked-shift", "scalar-bitwise", "scalar-comparison", "scalar-control-flow", "trap"} {
		if !slices.Contains(unit.Features, feature) {
			t.Errorf("features %v are missing %q", unit.Features, feature)
		}
	}
	for _, required := range []string{
		">= UINT8_C(8)",
		">= UINT32_C(32)",
		">= ((uintptr_t)UINT64_C(64))",
		") >> (UINT8_C(1))",
		") & (UINT32_C(63))",
		") ^ (((uintptr_t)UINT64_C(1)))",
		") | (((uintptr_t)UINT64_C(0)))",
		"(uint8_t)(~(uint8_t)(fy_l_",
		"(uint32_t)(~(uint32_t)(fy_l_",
		"(uintptr_t)(~(uintptr_t)(fy_l_",
	} {
		if !strings.Contains(unit.Source, required) {
			t.Errorf("generated C is missing typed unsigned operation %q:\n%s", required, unit.Source)
		}
	}
	if strings.Contains(unit.Source, "fy_sar_i32") || strings.Contains(unit.Source, "fy_bits_to_i32") {
		t.Fatalf("unsigned-only C unit includes signed bit helper machinery:\n%s", unit.Source)
	}
}

func TestEmitCheckedLeftShiftWithoutSignedCUndefinedBehavior(t *testing.T) {
	t.Parallel()
	program := lowerSource(t, `module main;
fn main() -> i32 {
    let signed_input: i32 = 21;
    let signed: i32 = signed_input << 1;
    let byte_input: u8 = 21;
    let byte: u8 = byte_input << 1;
    let word_input: u32 = 21;
    let word: u32 = word_input << 1;
    let size_input: usize = 21;
    let size: usize = size_input << 1;
    if signed == 42 && byte == 42 && word == 42 && size == 42 { 42 } else { 1 }
}
`)
	unit, err := cbackend.Emit(program, "main::main", cbackend.Stage0Profile())
	if err != nil {
		t.Fatalf("Emit() error = %v", err)
	}
	for _, feature := range []string{"checked-shift", "scalar-comparison", "scalar-control-flow", "trap"} {
		if !slices.Contains(unit.Features, feature) {
			t.Errorf("features %v are missing %q", unit.Features, feature)
		}
	}
	for _, required := range []string{
		"FY_TRAP_INVALID_SHIFT",
		"FY_TRAP_INTEGER_OVERFLOW",
		"(int64_t)INT32_MIN",
		"(int64_t)INT32_MAX",
		"UINT8_MAX >> (",
		"UINT32_MAX >> (",
		"UINTPTR_MAX >> (",
		"fy_bits_to_i32((uint32_t)(",
	} {
		if !strings.Contains(unit.Source, required) {
			t.Errorf("generated C is missing checked left shift %q:\n%s", required, unit.Source)
		}
	}
}

func TestEmitBorrowedViewEmptinessFromLengthMetadata(t *testing.T) {
	t.Parallel()
	program := lowerSource(t, `module main;
fn main() -> i32 {
    let text: &str = "fairy";
    let empty: &[u8] = b"";
    if !text.is_empty && empty.is_empty { 42 } else { 1 }
}
`)
	unit, err := cbackend.Emit(program, "main::main", cbackend.Stage0Profile())
	if err != nil {
		t.Fatalf("Emit() error = %v", err)
	}
	if !slices.Contains(unit.Features, "view-empty") {
		t.Fatalf("features %v are missing view-empty", unit.Features)
	}
	for _, required := range []string{
		".length == ((uintptr_t)UINT64_C(0))) ? INT32_C(1) : INT32_C(0)",
		"static const uint8_t fy_static_str_",
		"static const uint8_t fy_static_bytes_",
		"[1] = { UINT8_C(0) };",
	} {
		if !strings.Contains(unit.Source, required) {
			t.Errorf("generated C is missing view emptiness %q:\n%s", required, unit.Source)
		}
	}
}

func TestEmitByteStringLiteralDataAddressForForeignCall(t *testing.T) {
	t.Parallel()
	program := lowerSource(t, `module main;
@link_name("fairyc_fixture_is_fairy_bytes")
unsafe extern C fn is_fairy_bytes(value: *u8) -> u32;
fn main() -> i32 {
    let bytes: &[u8] = b"fairy";
    if unsafe { is_fairy_bytes(bytes as *u8) } == 1 { 42 } else { 1 }
}
`)
	unit, err := cbackend.Emit(program, "main::main", cbackend.Stage0Profile())
	if err != nil {
		t.Fatalf("Emit() error = %v", err)
	}
	for _, feature := range []string{"byte-slice-data-address", "foreign-c-call", "static-byte-string"} {
		if !slices.Contains(unit.Features, feature) {
			t.Errorf("features %v are missing %q", unit.Features, feature)
		}
	}
	if !slices.Equal(unit.ForeignSymbols, []string{"fairyc_fixture_is_fairy_bytes"}) {
		t.Fatalf("foreign symbols = %v", unit.ForeignSymbols)
	}
	for _, required := range []string{
		"extern uint32_t fairyc_fixture_is_fairy_bytes(const uint8_t * fy_a_0);",
		"static const uint8_t fy_static_bytes_",
		".length = ((uintptr_t)UINT64_C(5))",
		".data;",
		"fairyc_fixture_is_fairy_bytes(fy_l_",
	} {
		if !strings.Contains(unit.Source, required) {
			t.Errorf("generated C is missing %q:\n%s", required, unit.Source)
		}
	}
}

func TestEmitDistinctCIntForeignTransport(t *testing.T) {
	t.Parallel()
	program := lowerSource(t, `module main;
@link_name("fairyc_fixture_make_c_int")
unsafe extern C fn make_c_int() -> c::int;
@link_name("fairyc_fixture_consume_c_int")
unsafe extern C fn consume_c_int(value: c::int) -> i32;
fn main() -> i32 { unsafe { consume_c_int(make_c_int()) } }
`)
	_, interpreterErr := interpreter.Execute(program, "main::main")
	var interpreterCapability interpreter.CapabilityError
	if !errors.As(interpreterErr, &interpreterCapability) || interpreterCapability.Kind != "foreign-call" {
		t.Fatalf("interpreter error = %v", interpreterErr)
	}
	unit, err := cbackend.Emit(program, "main::main", cbackend.Stage0Profile())
	if err != nil {
		t.Fatalf("Emit() error = %v", err)
	}
	for _, required := range []string{
		"_Static_assert(sizeof(int) == UINT64_C(4), \"fairyc target c.int size\");",
		"_Static_assert(_Alignof(int) == UINT64_C(4), \"fairyc target c.int alignment\");",
		"extern int fairyc_fixture_make_c_int(void);",
		"extern int32_t fairyc_fixture_consume_c_int(int fy_a_0);",
		"fairyc_fixture_make_c_int();",
		"fairyc_fixture_consume_c_int(fy_l_",
	} {
		if !strings.Contains(unit.Source, required) {
			t.Errorf("generated C is missing %q:\n%s", required, unit.Source)
		}
	}
	if !slices.Contains(unit.Features, "c-int") || strings.Contains(unit.Source, "extern int32_t fairyc_fixture_make_c_int") || strings.Contains(unit.Source, "fairyc_fixture_consume_c_int(int32_t") {
		t.Fatalf("generated C erased c::int identity:\n%s", unit.Source)
	}
}

func TestEmitDistinctCSizeTForeignTransport(t *testing.T) {
	t.Parallel()
	program := lowerSource(t, `module main;
@link_name("fairyc_fixture_make_c_size_t")
unsafe extern C fn make_c_size_t() -> c::size_t;
@link_name("fairyc_fixture_consume_c_size_t")
unsafe extern C fn consume_c_size_t(value: c::size_t) -> i32;
fn main() -> i32 { unsafe { consume_c_size_t(make_c_size_t()) } }
`)
	unit, err := cbackend.Emit(program, "main::main", cbackend.Stage0Profile())
	if err != nil {
		t.Fatalf("Emit() error = %v", err)
	}
	for _, required := range []string{
		"#include <stddef.h>",
		"_Static_assert(sizeof(size_t) == UINT64_C(8), \"fairyc target c.size_t size\");",
		"_Static_assert(_Alignof(size_t) == UINT64_C(8), \"fairyc target c.size_t alignment\");",
		"_Static_assert(((size_t)-1) > (size_t)0, \"fairyc target c.size_t unsigned\");",
		"extern size_t fairyc_fixture_make_c_size_t(void);",
		"extern int32_t fairyc_fixture_consume_c_size_t(size_t fy_a_0);",
		"fairyc_fixture_consume_c_size_t(fy_l_",
	} {
		if !strings.Contains(unit.Source, required) {
			t.Errorf("generated C is missing %q:\n%s", required, unit.Source)
		}
	}
	if !slices.Contains(unit.Features, "c-size-t") || strings.Contains(unit.Source, "uintptr_t fairyc_fixture_make_c_size_t") || strings.Contains(unit.Source, "fairyc_fixture_consume_c_size_t(uintptr_t") {
		t.Fatalf("generated C erased c::size_t identity:\n%s", unit.Source)
	}
}

func TestEmitDistinctCLongForeignTransport(t *testing.T) {
	t.Parallel()
	program := lowerSource(t, `module main;
@link_name("fairyc_fixture_make_c_long")
unsafe extern C fn make_c_long() -> c::long;
@link_name("fairyc_fixture_consume_c_long")
unsafe extern C fn consume_c_long(value: c::long) -> i32;
fn main() -> i32 { unsafe { consume_c_long(make_c_long()) } }
`)
	unit, err := cbackend.Emit(program, "main::main", cbackend.Stage0Profile())
	if err != nil {
		t.Fatalf("Emit() error = %v", err)
	}
	for _, required := range []string{
		"_Static_assert(sizeof(long) == UINT64_C(8), \"fairyc target c.long size\");",
		"_Static_assert(_Alignof(long) == UINT64_C(8), \"fairyc target c.long alignment\");",
		"_Static_assert(((long)-1) < (long)0, \"fairyc target c.long signed\");",
		"extern long fairyc_fixture_make_c_long(void);",
		"extern int32_t fairyc_fixture_consume_c_long(long fy_a_0);",
		"fairyc_fixture_consume_c_long(fy_l_",
	} {
		if !strings.Contains(unit.Source, required) {
			t.Errorf("generated C is missing %q:\n%s", required, unit.Source)
		}
	}
	if !slices.Contains(unit.Features, "c-long") || strings.Contains(unit.Source, "int64_t fairyc_fixture_make_c_long") || strings.Contains(unit.Source, "fairyc_fixture_consume_c_long(int64_t") {
		t.Fatalf("generated C erased c::long identity:\n%s", unit.Source)
	}
}

func TestEmitDistinctCSSizeTForeignTransport(t *testing.T) {
	t.Parallel()
	program := lowerSource(t, `module main;
@link_name("fairyc_fixture_make_c_ssize_t")
unsafe extern C fn make_c_ssize_t() -> c::ssize_t;
@link_name("fairyc_fixture_consume_c_ssize_t")
unsafe extern C fn consume_c_ssize_t(value: c::ssize_t) -> i32;
fn main() -> i32 { unsafe { consume_c_ssize_t(make_c_ssize_t()) } }
`)
	unit, err := cbackend.Emit(program, "main::main", cbackend.Stage0Profile())
	if err != nil {
		t.Fatalf("Emit() error = %v", err)
	}
	for _, required := range []string{
		"#define _POSIX_C_SOURCE 200809L",
		"ISO C11 with POSIX.1-2008 typedefs",
		"#include <sys/types.h>",
		"_Static_assert(sizeof(ssize_t) == UINT64_C(8), \"fairyc target c.ssize_t size\");",
		"_Static_assert(_Alignof(ssize_t) == UINT64_C(8), \"fairyc target c.ssize_t alignment\");",
		"_Static_assert(((ssize_t)-1) < (ssize_t)0, \"fairyc target c.ssize_t signed\");",
		"extern ssize_t fairyc_fixture_make_c_ssize_t(void);",
		"extern int32_t fairyc_fixture_consume_c_ssize_t(ssize_t fy_a_0);",
		"fairyc_fixture_consume_c_ssize_t(fy_l_",
	} {
		if !strings.Contains(unit.Source, required) {
			t.Errorf("generated C is missing %q:\n%s", required, unit.Source)
		}
	}
	if !slices.Contains(unit.Features, "c-ssize-t") || strings.Contains(unit.Source, "intptr_t fairyc_fixture_make_c_ssize_t") || strings.Contains(unit.Source, "fairyc_fixture_consume_c_ssize_t(intptr_t") {
		t.Fatalf("generated C erased c::ssize_t identity:\n%s", unit.Source)
	}
}

func TestEmitRemainingCScalarForeignTransport(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name       string
		typeRef    string
		cType      string
		feature    string
		size       uint32
		alignment  uint32
		signedness string
	}{
		{name: "bool", typeRef: "c::bool", cType: "_Bool", feature: "c-bool", size: 1, alignment: 1},
		{name: "char", typeRef: "c::char", cType: "char", feature: "c-char", size: 1, alignment: 1, signedness: "signed"},
		{name: "double", typeRef: "c::double", cType: "double", feature: "c-double", size: 8, alignment: 8},
		{name: "float", typeRef: "c::float", cType: "float", feature: "c-float", size: 4, alignment: 4},
		{name: "longlong", typeRef: "c::longlong", cType: "long long", feature: "c-longlong", size: 8, alignment: 8, signedness: "signed"},
		{name: "schar", typeRef: "c::schar", cType: "signed char", feature: "c-schar", size: 1, alignment: 1, signedness: "signed"},
		{name: "short", typeRef: "c::short", cType: "short", feature: "c-short", size: 2, alignment: 2, signedness: "signed"},
		{name: "uchar", typeRef: "c::uchar", cType: "unsigned char", feature: "c-uchar", size: 1, alignment: 1, signedness: "unsigned"},
		{name: "uint", typeRef: "c::uint", cType: "unsigned int", feature: "c-uint", size: 4, alignment: 4, signedness: "unsigned"},
		{name: "ulong", typeRef: "c::ulong", cType: "unsigned long", feature: "c-ulong", size: 8, alignment: 8, signedness: "unsigned"},
		{name: "ulonglong", typeRef: "c::ulonglong", cType: "unsigned long long", feature: "c-ulonglong", size: 8, alignment: 8, signedness: "unsigned"},
		{name: "ushort", typeRef: "c::ushort", cType: "unsigned short", feature: "c-ushort", size: 2, alignment: 2, signedness: "unsigned"},
	}
	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			program := lowerSource(t, fmt.Sprintf(`module main;
@link_name("fairyc_fixture_make_c_%[1]s")
unsafe extern C fn make_%[1]s() -> %[2]s;
@link_name("fairyc_fixture_consume_c_%[1]s")
unsafe extern C fn consume_%[1]s(value: %[2]s) -> i32;
fn main() -> i32 { unsafe { consume_%[1]s(make_%[1]s()) } }
`, test.name, test.typeRef))
			unit, err := cbackend.Emit(program, "main::main", cbackend.Stage0Profile())
			if err != nil {
				t.Fatalf("Emit() error = %v", err)
			}
			targetName := strings.Replace(test.typeRef, "::", ".", 1)
			required := []string{
				fmt.Sprintf("_Static_assert(sizeof(%s) == UINT64_C(%d), \"fairyc target %s size\");", test.cType, test.size, targetName),
				fmt.Sprintf("_Static_assert(_Alignof(%s) == UINT64_C(%d), \"fairyc target %s alignment\");", test.cType, test.alignment, targetName),
				fmt.Sprintf("extern %s fairyc_fixture_make_c_%s(void);", test.cType, test.name),
				fmt.Sprintf("extern int32_t fairyc_fixture_consume_c_%s(%s fy_a_0);", test.name, test.cType),
				"fairyc_fixture_consume_c_" + test.name + "(fy_l_",
			}
			if test.signedness != "" {
				operator := "<"
				if test.signedness == "unsigned" {
					operator = ">"
				}
				required = append(required, fmt.Sprintf("_Static_assert(((%s)-1) %s (%s)0, \"fairyc target %s %s\");", test.cType, operator, test.cType, targetName, test.signedness))
			}
			for _, text := range required {
				if !strings.Contains(unit.Source, text) {
					t.Errorf("generated C is missing %q:\n%s", text, unit.Source)
				}
			}
			if !slices.Contains(unit.Features, test.feature) {
				t.Fatalf("features = %v, want %q", unit.Features, test.feature)
			}
		})
	}
}

func TestEmitKeepsCScalarPointersOutsideByValueTransportSlice(t *testing.T) {
	t.Parallel()
	for _, cScalar := range hir.CScalarTypes() {
		typeRef := string(cScalar)
		t.Run(typeRef, func(t *testing.T) {
			program := lowerSource(t, "module main;\nunsafe extern C fn consume_pointer(value: *"+typeRef+") -> i32;\nfn main() -> i32 { unsafe { consume_pointer(null) } }\n")
			_, err := cbackend.Emit(program, "main::main", cbackend.Stage0Profile())
			var capability cbackend.CapabilityError
			if !errors.As(err, &capability) || !strings.Contains(capability.Feature, "foreign raw-pointer element type \""+typeRef+"\"") {
				t.Fatalf("Emit() error = %v", err)
			}
		})
	}
}

func TestEmitRawPointerAndTargetScalarForeignCall(t *testing.T) {
	t.Parallel()
	program := lowerSource(t, "module main;\nunsafe extern C fn foreign_probe(data: *u8, output: *mut u32, length: usize, marker: u8) -> u32;\nfn main() -> i32 { if unsafe { foreign_probe(null, null, 0, 7) } == 42 { 0 } else { 1 } }\n")
	unit, err := cbackend.Emit(program, "main::main", cbackend.Stage0Profile())
	if err != nil {
		t.Fatalf("Emit() error = %v", err)
	}
	for _, required := range []string{
		"extern uint32_t foreign_probe(const uint8_t * fy_a_0, uint32_t * fy_a_1, uintptr_t fy_a_2, uint8_t fy_a_3);",
		"foreign_probe(((void *)0), ((void *)0), ((uintptr_t)UINT64_C(0)), UINT8_C(7));",
		"== (UINT32_C(42))",
	} {
		if !strings.Contains(unit.Source, required) {
			t.Errorf("generated C is missing %q:\n%s", required, unit.Source)
		}
	}
}

func TestEmitMutableVoidPointerOutputSlot(t *testing.T) {
	t.Parallel()
	program := lowerSource(t, `module main;
unsafe extern C fn fill(output: *mut *mut void) -> u32;
fn main() -> i32 {
    var output: *mut void = null;
    if unsafe { fill(&mut output as *mut *mut void) } == 0 { 42 } else { 1 }
}
`)
	unit, err := cbackend.Emit(program, "main::main", cbackend.Stage0Profile())
	if err != nil {
		t.Fatalf("Emit() error = %v", err)
	}
	for _, required := range []string{
		"extern uint32_t fill(void * * fy_a_0);",
		"void * fy_l_",
		"void * * fy_l_",
		" = &fy_l_",
		"fill(fy_l_",
	} {
		if !strings.Contains(unit.Source, required) {
			t.Errorf("generated C is missing %q:\n%s", required, unit.Source)
		}
	}
}

func TestEmitOpaqueAddressOfMutableLocal(t *testing.T) {
	t.Parallel()
	program := lowerSource(t, `module main;
@repr(C)
struct Packet { code: u32; }
unsafe extern C fn sink(output: *mut void) -> u32;
fn main() -> i32 {
    var output: Packet = Packet { code: 0 };
    if unsafe { sink(&mut output as *mut void) } == 0 { 42 } else { 1 }
}
`)
	unit, err := cbackend.Emit(program, "main::main", cbackend.Stage0Profile())
	if err != nil {
		t.Fatalf("Emit() error = %v", err)
	}
	for _, required := range []string{
		"extern uint32_t sink(void * fy_a_0);",
		"void * fy_l_",
		" = &fy_l_",
		"sink(fy_l_",
	} {
		if !strings.Contains(unit.Source, required) {
			t.Errorf("generated C is missing %q:\n%s", required, unit.Source)
		}
	}
}

func TestEmitCStrParameterDataAddress(t *testing.T) {
	t.Parallel()
	program := lowerSource(t, `module main;
unsafe extern C fn inspect_raw(path: *u8) -> u32;
fn inspect(path: &CStr) -> u32 { unsafe { inspect_raw(path as *u8) } }
fn main() -> i32 { if inspect(c"fairy") == 0 { 42 } else { 1 } }
`)
	unit, err := cbackend.Emit(program, "main::main", cbackend.Stage0Profile())
	if err != nil {
		t.Fatalf("Emit() error = %v", err)
	}
	for _, required := range []string{
		"extern uint32_t inspect_raw(const uint8_t * fy_a_0);",
		"const uint8_t * fy_l_",
		" = fy_l_",
		"inspect_raw(fy_l_",
	} {
		if !strings.Contains(unit.Source, required) {
			t.Errorf("generated C is missing %q:\n%s", required, unit.Source)
		}
	}
	if strings.Contains(unit.Source, ".data;") {
		t.Fatalf("generated C treats a CStr pointer parameter as a descriptor:\n%s", unit.Source)
	}
}

func TestEmitResourceAndArraySafeBorrowsToOpaquePointers(t *testing.T) {
	t.Parallel()
	program := lowerSource(t, `module main;
resource Token { value: u32; drop fn(self: &mut Self) {} }
unsafe extern C fn inspect_raw(token: *mut void, bytes: *mut void) -> u32;
fn inspect(token: &mut Token, bytes: &mut [u8; 4]) -> u32 {
    unsafe { inspect_raw(token as *mut void, bytes as *mut void) }
}
fn main() -> i32 {
    var token: Token = Token { value: 42 };
    var bytes: [u8; 4] = [0, 0, 0, 0];
    if inspect(&mut token, &mut bytes) == 0 { 42 } else { 1 }
}
`)
	unit, err := cbackend.Emit(program, "main::main", cbackend.Stage0Profile())
	if err != nil {
		t.Fatalf("Emit() error = %v", err)
	}
	for _, feature := range []string{"raw-borrow-pointer", "scalar-fixed-array", "flat-copy-resource"} {
		if !slices.Contains(unit.Features, feature) {
			t.Errorf("features = %v, want %q", unit.Features, feature)
		}
	}
	for _, required := range []string{
		"extern uint32_t inspect_raw(void * fy_a_0, void * fy_a_1);",
		"fy_resource_0 *",
		"fy_array_0 *",
		" = &fy_l_",
		"inspect_raw(fy_l_",
	} {
		if !strings.Contains(unit.Source, required) {
			t.Errorf("generated C is missing %q:\n%s", required, unit.Source)
		}
	}
}

func TestEmitRejectsOtherNestedRawPointerMutabilityShapes(t *testing.T) {
	t.Parallel()
	for _, typeRef := range []string{"**void", "**mut void", "*mut *void"} {
		t.Run(typeRef, func(t *testing.T) {
			program := lowerSource(t, "module main;\nunsafe extern C fn inspect(value: "+typeRef+") -> u32;\nfn main() -> i32 { if unsafe { inspect(null) } == 0 { 42 } else { 1 } }\n")
			_, err := cbackend.Emit(program, "main::main", cbackend.Stage0Profile())
			var capability cbackend.CapabilityError
			if !errors.As(err, &capability) || !strings.Contains(capability.Feature, "foreign raw-pointer nesting type") {
				t.Fatalf("Emit() error = %v", err)
			}
		})
	}
}

func TestEmitRejectsReservedExternalSymbol(t *testing.T) {
	t.Parallel()
	program := lowerSource(t, "module main;\n@link_name(\"main\")\nunsafe extern C fn add_one(value: i32) -> i32;\nfn main() -> i32 { unsafe { add_one(41) } }\n")
	_, err := cbackend.Emit(program, "main::main", cbackend.Stage0Profile())
	var capability cbackend.CapabilityError
	if !errors.As(err, &capability) || capability.Function != "main::add_one" || capability.Feature != "external C symbol \"main\"" {
		t.Fatalf("Emit() error = %v", err)
	}
}

func TestEmitRejectsUnrecognizedStandardLibrarySymbol(t *testing.T) {
	t.Parallel()
	program := lowerSource(t, "module main;\n@link_name(\"fy_std_unrecognized_v0\")\nunsafe extern C fn add_one(value: i32) -> i32;\nfn main() -> i32 { unsafe { add_one(41) } }\n")
	_, err := cbackend.Emit(program, "main::main", cbackend.Stage0Profile())
	var capability cbackend.CapabilityError
	if !errors.As(err, &capability) || capability.Function != "main::add_one" || capability.Feature != "external C symbol \"fy_std_unrecognized_v0\"" {
		t.Fatalf("Emit() error = %v", err)
	}
}

func TestEmitKeepsUnexercisedLinuxFileSymbolsClosed(t *testing.T) {
	t.Parallel()
	for _, symbol := range []string{"fy_std_linux_file_write_v0", "fy_std_linux_file_drop_v0"} {
		symbol := symbol
		t.Run(symbol, func(t *testing.T) {
			t.Parallel()
			program := lowerSource(t, "module main;\n@link_name(\""+symbol+"\")\nunsafe extern C fn file_operation(value: i32) -> i32;\nfn main() -> i32 { unsafe { file_operation(41) } }\n")
			_, err := cbackend.Emit(program, "main::main", cbackend.Stage0Profile())
			var capability cbackend.CapabilityError
			if !errors.As(err, &capability) || capability.Function != "main::file_operation" || capability.Feature != "external C symbol \""+symbol+"\"" {
				t.Fatalf("Emit() error = %v", err)
			}
		})
	}
}

func TestEmitRejectsLinkNameThatIsNotAC11Identifier(t *testing.T) {
	t.Parallel()
	program := lowerSource(t, "module main;\n@link_name(\"bad-name\")\nunsafe extern C fn add_one(value: i32) -> i32;\nfn main() -> i32 { unsafe { add_one(41) } }\n")
	_, err := cbackend.Emit(program, "main::main", cbackend.Stage0Profile())
	var capability cbackend.CapabilityError
	if !errors.As(err, &capability) || capability.Function != "main::add_one" || capability.Feature != "external C symbol \"bad-name\"" {
		t.Fatalf("Emit() error = %v", err)
	}
}

func TestEmitGuardsExternalSymbolAgainstHeaderMacro(t *testing.T) {
	t.Parallel()
	program := lowerSource(t, "module main;\nunsafe extern C fn INT64_MIN(value: i32) -> i32;\nfn main() -> i32 { unsafe { INT64_MIN(41) } }\n")
	unit, err := cbackend.Emit(program, "main::main", cbackend.Stage0Profile())
	if err != nil {
		t.Fatalf("Emit() error = %v", err)
	}
	for _, required := range []string{
		"#ifdef INT64_MIN\n#undef INT64_MIN\n#endif",
		"extern int32_t INT64_MIN(int32_t fy_a_0);",
		"INT64_MIN(INT32_C(41));",
	} {
		if !strings.Contains(unit.Source, required) {
			t.Errorf("generated C is missing %q:\n%s", required, unit.Source)
		}
	}
}

func TestEmitCoalescesCompatibleExternalDeclarations(t *testing.T) {
	t.Parallel()
	program := lowerSources(t, map[string]string{
		"bindingsa.fy": "module bindingsa;\n@link_name(\"shared_add_one\")\npub unsafe extern C fn add_from_a(value: i32) -> i32;\n",
		"bindingsb.fy": "module bindingsb;\n@link_name(\"shared_add_one\")\npub unsafe extern C fn add_from_b(value: i32) -> i32;\n",
		"main.fy":      "module main;\nimport bindingsb as b;\nfn main() -> i32 { unsafe { b::add_from_b(41) } }\n",
	})
	unit, err := cbackend.Emit(program, "main::main", cbackend.Stage0Profile())
	if err != nil {
		t.Fatalf("Emit() error = %v", err)
	}
	prototype := "extern int32_t shared_add_one(int32_t fy_a_0);"
	if count := strings.Count(unit.Source, prototype); count != 1 {
		t.Fatalf("generated C contains %d compatible prototypes, want 1:\n%s", count, unit.Source)
	}
}

func TestEmitRejectsConflictingExternalDeclarations(t *testing.T) {
	t.Parallel()
	program := lowerSources(t, map[string]string{
		"bindingsa.fy": "module bindingsa;\n@link_name(\"shared_symbol\")\npub unsafe extern C fn accepts_value(value: i32) -> i32;\n",
		"bindingsb.fy": "module bindingsb;\n@link_name(\"shared_symbol\")\npub unsafe extern C fn no_arguments() -> i32;\n",
		"main.fy":      "module main;\nfn main() -> i32 { 0 }\n",
	})
	_, err := cbackend.Emit(program, "main::main", cbackend.Stage0Profile())
	if err == nil || !strings.Contains(err.Error(), "C emission requires verified MIR: V1001") {
		t.Fatalf("Emit() error = %v", err)
	}
}

func TestTrapCodesAreStable(t *testing.T) {
	t.Parallel()
	want := map[string]uint32{
		"bounds":                   1,
		"integer-overflow":         2,
		"division-by-zero":         3,
		"signed-division-overflow": 4,
		"invalid-shift":            5,
		"assertion-failed":         6,
		"unreachable":              7,
	}
	for category, expected := range want {
		actual, ok := cbackend.TrapCode(category)
		if !ok || actual != expected {
			t.Errorf("TrapCode(%q) = %d, %t; want %d, true", category, actual, ok, expected)
		}
	}
	if code, ok := cbackend.TrapCode("unknown"); ok || code != 0 {
		t.Fatalf("TrapCode(unknown) = %d, %t", code, ok)
	}
}

func TestEmitMaterializesCheckedDivisionDivisor(t *testing.T) {
	t.Parallel()
	for _, testCase := range []struct {
		name     string
		operator string
	}{
		{name: "division", operator: "/"},
		{name: "remainder", operator: "%"},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			program := lowerSource(t, fmt.Sprintf("module main;\nfn main() -> i32 { 42 %s 0 }\n", testCase.operator))
			unit, err := cbackend.Emit(program, "main::main", cbackend.Stage0Profile())
			if err != nil {
				t.Fatalf("Emit() error = %v", err)
			}
			for _, required := range []string{
				"volatile int32_t fy_checked_divisor_i32;",
				"fy_checked_divisor_i32 = INT32_C(0);",
				fmt.Sprintf("(INT32_C(42)) %s fy_checked_divisor_i32;", testCase.operator),
			} {
				if !strings.Contains(unit.Source, required) {
					t.Errorf("generated C is missing %q:\n%s", required, unit.Source)
				}
			}
			if strings.Contains(unit.Source, fmt.Sprintf("%s (INT32_C(0))", testCase.operator)) {
				t.Fatalf("generated C exposes a checked zero divisor as a constant C expression:\n%s", unit.Source)
			}
		})
	}
}

func TestEmitConstructsCheckedArithmeticWithoutSignedCOperations(t *testing.T) {
	t.Parallel()
	program := lowerSource(t, `module main;
fn main() -> i32 {
    let added: i32 = 2147483647 + 1;
    -added
}
`)
	unit, err := cbackend.Emit(program, "main::main", cbackend.Stage0Profile())
	if err != nil {
		t.Fatalf("Emit() error = %v", err)
	}
	for _, required := range []string{
		"static int32_t fy_bits_to_i32(uint32_t bits)",
		"fy_bits_to_i32((uint32_t)(INT32_C(2147483647)) + (uint32_t)(INT32_C(1)))",
		"fy_bits_to_i32(UINT32_C(0) - (uint32_t)(fy_l_",
	} {
		if !strings.Contains(unit.Source, required) {
			t.Errorf("generated C is missing %q:\n%s", required, unit.Source)
		}
	}
	if strings.Contains(unit.Source, "(INT32_C(2147483647)) + (INT32_C(1))") {
		t.Fatalf("generated C exposes checked signed overflow as a constant C expression:\n%s", unit.Source)
	}
}

func TestEmitFlatI32Resource(t *testing.T) {
	t.Parallel()
	program := lowerSource(t, `module main;
resource Pair {
    left: i32;
    right: i32;
    drop fn(self: &mut Self) {}
}
fn transfer(pair: Pair) -> Pair { pair }
fn main() -> i32 {
    let pair: Pair = Pair { left: 1, right: 42 };
    let moved: Pair = transfer(pair);
    moved.right
}
`)
	outcome, err := interpreter.Execute(program, "main::main")
	if err != nil || outcome.Kind != interpreter.OutcomeReturn || outcome.Value != 42 {
		t.Fatalf("interpreter = %+v, %v", outcome, err)
	}
	unit, err := cbackend.Emit(program, "main::main", cbackend.Stage0Profile())
	if err != nil {
		t.Fatalf("Emit() error = %v", err)
	}
	wantFeatures := []string{"flat-i32-resource", "scalar-call"}
	if !slices.Equal(unit.Features, wantFeatures) {
		t.Fatalf("Emit() features = %v, want %v", unit.Features, wantFeatures)
	}
	for _, required := range []string{
		"#include <stddef.h>",
		"} fy_resource_0;",
		"_Static_assert(sizeof(fy_resource_0) == UINT64_C(8)",
		"_Static_assert(_Alignof(fy_resource_0) == UINT64_C(4)",
		"offsetof(fy_resource_0, f_0) == UINT64_C(0)",
		"offsetof(fy_resource_0, f_1) == UINT64_C(4)",
		"(fy_resource_0){ .f_0 = INT32_C(1), .f_1 = INT32_C(42) }",
		".f_1;",
	} {
		if !strings.Contains(unit.Source, required) {
			t.Errorf("generated C is missing %q", required)
		}
	}
	for _, leaked := range []string{"Pair", "left", "right"} {
		if strings.Contains(unit.Source, leaked) {
			t.Errorf("generated C leaks source resource identifier %q", leaked)
		}
	}
}

func TestEmitFlatCopyResourceWithRawPointer(t *testing.T) {
	t.Parallel()
	program := lowerSource(t, `module main;
resource Handle {
    raw: *mut void;
    state: u32;
    drop fn(self: &mut Self) {}
}
fn main() -> i32 {
    let handle: Handle = Handle { raw: null, state: 42 };
    if handle.raw == null {
        if handle.state == 42 { 42 } else { 1 }
    } else {
        2
    }
}
`)
	outcome, err := interpreter.Execute(program, "main::main")
	if err != nil || outcome.Kind != interpreter.OutcomeReturn || outcome.Value != 42 {
		t.Fatalf("interpreter = %+v, %v", outcome, err)
	}
	unit, err := cbackend.Emit(program, "main::main", cbackend.Stage0Profile())
	if err != nil {
		t.Fatalf("Emit() error = %v", err)
	}
	wantFeatures := []string{"flat-copy-resource", "scalar-comparison", "scalar-control-flow"}
	if !slices.Equal(unit.Features, wantFeatures) {
		t.Fatalf("Emit() features = %v, want %v", unit.Features, wantFeatures)
	}
	for _, required := range []string{
		"void * f_0;",
		"uint32_t f_1;",
		"_Static_assert(sizeof(fy_resource_0) == UINT64_C(16)",
		"_Static_assert(_Alignof(fy_resource_0) == UINT64_C(8)",
		"offsetof(fy_resource_0, f_0) == UINT64_C(0)",
		"offsetof(fy_resource_0, f_1) == UINT64_C(8)",
		"(fy_resource_0){ .f_0 = ((void *)0), .f_1 = UINT32_C(42) }",
		".f_0;",
		".f_1;",
	} {
		if !strings.Contains(unit.Source, required) {
			t.Errorf("generated C is missing %q", required)
		}
	}
}

func TestEmitRejectsNestedStructResourceField(t *testing.T) {
	t.Parallel()
	program := lowerSource(t, "module main;\nstruct Marker { value: u32; }\nresource Holder { marker: Marker; drop fn(self: &mut Self) {} }\nfn main() -> i32 { 0 }\n")
	_, err := cbackend.Emit(program, "main::main", cbackend.Stage0Profile())
	var capability cbackend.CapabilityError
	if !errors.As(err, &capability) || !strings.Contains(capability.Feature, "field \"marker\" of type \"main::Marker\"") || capability.Function != "" {
		t.Fatalf("Emit() error = %v", err)
	}
}

func TestEmitReprCStructPreservesPublicTagFieldsAndLayout(t *testing.T) {
	t.Parallel()
	program := lowerSource(t, `module main;
@repr(C)
struct fy_std_text_view_v0 { data: *u8; length: usize; }
unsafe extern C fn foreign_take(output: *mut fy_std_text_view_v0) -> u32;
fn main() -> i32 {
    var view: fy_std_text_view_v0 = fy_std_text_view_v0 { data: null, length: 0 };
    if unsafe { foreign_take(&mut view as *mut fy_std_text_view_v0) } == 0 { 0 } else { 1 }
}
`)
	unit, err := cbackend.Emit(program, "main::main", cbackend.Stage0Profile())
	if err != nil {
		t.Fatalf("Emit() error = %v", err)
	}
	for _, required := range []string{
		"struct fy_std_text_view_v0 {",
		"const uint8_t * data;",
		"uintptr_t length;",
		"_Static_assert(sizeof(struct fy_std_text_view_v0) == UINT64_C(16)",
		"offsetof(struct fy_std_text_view_v0, data) == UINT64_C(0)",
		"offsetof(struct fy_std_text_view_v0, length) == UINT64_C(8)",
		"extern uint32_t foreign_take(struct fy_std_text_view_v0 *",
		"(struct fy_std_text_view_v0){ .data = ((void *)0)",
		".length = ((uintptr_t)UINT64_C(0)) }",
		" = &fy_l_",
		"foreign_take(fy_l_",
	} {
		if !strings.Contains(unit.Source, required) {
			t.Errorf("generated C is missing %q:\n%s", required, unit.Source)
		}
	}
	if !slices.Contains(unit.Features, "inert-struct") || !slices.Contains(unit.Features, "repr-c-struct") {
		t.Fatalf("features = %v", unit.Features)
	}
}

func TestEmitReprCStructForeignResult(t *testing.T) {
	t.Parallel()
	program := lowerSource(t, `module main;
@repr(C)
struct fairyc_fixture_status {
    status: u32;
    detail: u32;
}
@link_name("fairyc_fixture_make_status")
unsafe extern C fn make_status() -> fairyc_fixture_status;
fn main() -> i32 {
    let result: fairyc_fixture_status = unsafe { make_status() };
    if result.status == 41 && result.detail == 1 { 42 } else { 0 }
}
`)
	unit, err := cbackend.Emit(program, "main::main", cbackend.Stage0Profile())
	if err != nil {
		t.Fatalf("Emit() error = %v", err)
	}
	if !slices.Contains(unit.Features, "foreign-c-struct-result") {
		t.Fatalf("features = %v", unit.Features)
	}
	for _, required := range []string{
		"struct fairyc_fixture_status {",
		"uint32_t status;",
		"uint32_t detail;",
		"_Static_assert(sizeof(struct fairyc_fixture_status) == UINT64_C(8)",
		"extern struct fairyc_fixture_status fairyc_fixture_make_status(void);",
		" = fairyc_fixture_make_status();",
		".status;",
		".detail;",
	} {
		if !strings.Contains(unit.Source, required) {
			t.Errorf("generated C is missing %q:\n%s", required, unit.Source)
		}
	}
}

func TestEmitRejectsReprCStructForeignParameter(t *testing.T) {
	t.Parallel()
	program := lowerSource(t, `module main;
@repr(C)
struct fairyc_fixture_status { status: u32; detail: u32; }
unsafe extern C fn consume_status(value: fairyc_fixture_status) -> i32;
fn main() -> i32 {
    let value: fairyc_fixture_status = fairyc_fixture_status { status: 41, detail: 1 };
    unsafe { consume_status(value) }
}
`)
	_, err := cbackend.Emit(program, "main::main", cbackend.Stage0Profile())
	var capability cbackend.CapabilityError
	if !errors.As(err, &capability) || capability.Function != "main::consume_status" || capability.Feature != `foreign parameter type "main::fairyc_fixture_status"` {
		t.Fatalf("Emit() error = %v", err)
	}
}

func TestEmitRejectsNonReprCStructForeignResult(t *testing.T) {
	t.Parallel()
	program := lowerSource(t, `module main;
struct PrivateStatus { status: u32; detail: u32; }
unsafe extern C fn make_status() -> PrivateStatus;
fn main() -> i32 {
    let result: PrivateStatus = unsafe { make_status() };
    if result.status == 41 { 42 } else { 0 }
}
`)
	_, err := cbackend.Emit(program, "main::main", cbackend.Stage0Profile())
	var capability cbackend.CapabilityError
	if !errors.As(err, &capability) || capability.Function != "main::make_status" || capability.Feature != `foreign result struct "main::PrivateStatus" is not repr(C)` {
		t.Fatalf("Emit() error = %v", err)
	}
}

func TestEmitReprCStructPreservesVoidPointerAndExactCScalarFields(t *testing.T) {
	t.Parallel()
	program := lowerSource(t, `module main;
@repr(C)
struct fy_std_system_heap_block_v0 {
    data: *mut void;
    size: c::size_t;
    alignment: c::size_t;
}
fn main() -> i32 { 0 }
`)
	unit, err := cbackend.Emit(program, "main::main", cbackend.Stage0Profile())
	if err != nil {
		t.Fatalf("Emit() error = %v", err)
	}
	for _, required := range []string{
		"_Static_assert(sizeof(size_t) == UINT64_C(8), \"fairyc target c.size_t size\");",
		"struct fy_std_system_heap_block_v0 {",
		"void * data;",
		"size_t size;",
		"size_t alignment;",
		"_Static_assert(sizeof(struct fy_std_system_heap_block_v0) == UINT64_C(24)",
		"offsetof(struct fy_std_system_heap_block_v0, data) == UINT64_C(0)",
		"offsetof(struct fy_std_system_heap_block_v0, size) == UINT64_C(8)",
		"offsetof(struct fy_std_system_heap_block_v0, alignment) == UINT64_C(16)",
	} {
		if !strings.Contains(unit.Source, required) {
			t.Errorf("generated C is missing %q:\n%s", required, unit.Source)
		}
	}
	for _, feature := range []string{"c-size-t", "inert-struct", "repr-c-struct"} {
		if !slices.Contains(unit.Features, feature) {
			t.Fatalf("features = %v, want %q", unit.Features, feature)
		}
	}
}

func TestEmitNestedReprCStructsInDependencyOrderWithExactLayout(t *testing.T) {
	t.Parallel()
	program := lowerSource(t, `module main;
@repr(C)
struct fy_std_buffer_v0 {
    block: fy_std_system_heap_block_v0;
    length: c::size_t;
}
@repr(C)
struct fy_std_system_heap_block_v0 {
    data: *mut void;
    size: c::size_t;
    alignment: c::size_t;
}
fn main() -> i32 { 0 }
`)
	unit, err := cbackend.Emit(program, "main::main", cbackend.Stage0Profile())
	if err != nil {
		t.Fatalf("Emit() error = %v", err)
	}
	heapDefinition := strings.Index(unit.Source, "struct fy_std_system_heap_block_v0 {")
	bufferDefinition := strings.Index(unit.Source, "struct fy_std_buffer_v0 {")
	if heapDefinition < 0 || bufferDefinition < 0 || heapDefinition >= bufferDefinition {
		t.Fatalf("nested repr(C) definitions are not dependency ordered:\n%s", unit.Source)
	}
	for _, required := range []string{
		"struct fy_std_system_heap_block_v0 block;",
		"size_t length;",
		"_Static_assert(sizeof(struct fy_std_system_heap_block_v0) == UINT64_C(24)",
		"offsetof(struct fy_std_system_heap_block_v0, data) == UINT64_C(0)",
		"offsetof(struct fy_std_system_heap_block_v0, size) == UINT64_C(8)",
		"offsetof(struct fy_std_system_heap_block_v0, alignment) == UINT64_C(16)",
		"_Static_assert(sizeof(struct fy_std_buffer_v0) == UINT64_C(32)",
		"offsetof(struct fy_std_buffer_v0, block) == UINT64_C(0)",
		"offsetof(struct fy_std_buffer_v0, length) == UINT64_C(24)",
	} {
		if !strings.Contains(unit.Source, required) {
			t.Errorf("generated C is missing %q:\n%s", required, unit.Source)
		}
	}
	for _, feature := range []string{"c-size-t", "inert-struct", "repr-c-struct"} {
		if !slices.Contains(unit.Features, feature) {
			t.Fatalf("features = %v, want %q", unit.Features, feature)
		}
	}
}

func TestEmitRejectsRecursiveStructDependency(t *testing.T) {
	t.Parallel()
	program := lowerSource(t, `module main;
@repr(C)
struct First { second: Second; }
@repr(C)
struct Second { first: First; }
fn main() -> i32 { 0 }
`)
	_, err := cbackend.Emit(program, "main::main", cbackend.Stage0Profile())
	var capability cbackend.CapabilityError
	if !errors.As(err, &capability) || !strings.Contains(capability.Feature, "recursive struct dependency involving") {
		t.Fatalf("Emit() error = %v", err)
	}
}

func TestEmitRejectsNonReprCNestedStruct(t *testing.T) {
	t.Parallel()
	program := lowerSource(t, `module main;
struct Private { value: u32; }
@repr(C)
struct Public { inner: Private; }
fn main() -> i32 { 0 }
`)
	_, err := cbackend.Emit(program, "main::main", cbackend.Stage0Profile())
	var capability cbackend.CapabilityError
	if !errors.As(err, &capability) || !strings.Contains(capability.Feature, `struct type "main::Public" field "inner" of type "main::Private": nested struct type "main::Private" is not repr(C)`) {
		t.Fatalf("Emit() error = %v", err)
	}
}

func TestEmitKeepsCScalarFieldsInsideReprCStructs(t *testing.T) {
	t.Parallel()
	program := lowerSource(t, "module main;\nstruct Private { size: c::size_t; }\nfn main() -> i32 { 0 }\n")
	_, err := cbackend.Emit(program, "main::main", cbackend.Stage0Profile())
	var capability cbackend.CapabilityError
	if !errors.As(err, &capability) || !strings.Contains(capability.Feature, "struct type \"main::Private\" field \"size\" of type \"c::size_t\"") {
		t.Fatalf("Emit() error = %v", err)
	}
}

func TestEmitRestrictedEnumConstructionPayloadAndSwitch(t *testing.T) {
	t.Parallel()
	program := lowerSource(t, `module main;
enum Message {
    Empty;
    Value { amount: i32; };
    Pair { left: i32; right: i32; };
}
fn score(message: Message) -> i32 {
    match message {
        Message::Empty => 0,
        Message::Value { amount: value } => value,
        Message::Pair { left: first, right: second } => first + second,
    }
}
fn main() -> i32 { score(Message::Pair { left: 20, right: 22 }) }
`)
	unit, err := cbackend.Emit(program, "main::main", cbackend.Stage0Profile())
	if err != nil {
		t.Fatalf("Emit() error = %v", err)
	}
	wantFeatures := []string{"checked-arithmetic", "inert-enum", "scalar-call", "scalar-control-flow", "trap"}
	if !slices.Equal(unit.Features, wantFeatures) {
		t.Fatalf("features = %v, want %v", unit.Features, wantFeatures)
	}
	for _, required := range []string{
		"typedef union {\n    uint8_t unused;",
		"struct {\n        int32_t f_0;\n        int32_t f_1;\n    } v_2;",
		"typedef struct { uint32_t tag; fy_enum_payload_0 payload; } fy_enum_0;",
		"(fy_enum_0){ .tag = UINT32_C(2), .payload.v_2 = { .f_0 = INT32_C(20), .f_1 = INT32_C(22) } }",
		".tag == UINT32_C(0)",
		".payload.v_2.f_0;",
		"fy_runtime_trap(FY_TRAP_UNREACHABLE);",
	} {
		if !strings.Contains(unit.Source, required) {
			t.Errorf("generated C is missing %q:\n%s", required, unit.Source)
		}
	}
	for _, leaked := range []string{"Message", "Empty", "Value", "Pair", "amount", "left", "right"} {
		if strings.Contains(unit.Source, leaked) {
			t.Errorf("generated C leaks source enum identifier %q", leaked)
		}
	}
}

func TestEmitRejectsNestedEnumPayloadUntilDependencyOrderingExists(t *testing.T) {
	t.Parallel()
	program := lowerSource(t, "module main;\nenum Inner { Ready; }\nenum Outer { Wrapped { value: Inner; }; }\nfn main() -> i32 { 0 }\n")
	_, err := cbackend.Emit(program, "main::main", cbackend.Stage0Profile())
	var capability cbackend.CapabilityError
	if !errors.As(err, &capability) || !strings.Contains(capability.Feature, "enum type \"main::Outer\"") || !strings.Contains(capability.Feature, "field \"value\" of type \"main::Inner\"") {
		t.Fatalf("Emit() error = %v", err)
	}
}

func TestEmitRejectsInvalidReprCStructName(t *testing.T) {
	t.Parallel()
	program := lowerSource(t, "module main;\n@repr(C)\nstruct Packet { value: u32; }\nfn main() -> i32 { 0 }\n")
	program.StructTypes[0].Name = "bad-name"
	_, err := cbackend.Emit(program, "main::main", cbackend.Stage0Profile())
	var capability cbackend.CapabilityError
	if !errors.As(err, &capability) || capability.Feature != "repr(C) struct name \"bad-name\"" {
		t.Fatalf("Emit() error = %v", err)
	}
}

func TestEmitRejectsNonScalarParameterType(t *testing.T) {
	t.Parallel()
	program := lowerSource(t, "module main;\nfn inspect(value: &i32) -> i32 { 0 }\nfn main() -> i32 { 0 }\n")
	_, err := cbackend.Emit(program, "main::main", cbackend.Stage0Profile())
	var capability cbackend.CapabilityError
	if !errors.As(err, &capability) || capability.Feature != "parameter type \"&i32\"" || capability.Function != "main::inspect" {
		t.Fatalf("Emit() error = %v", err)
	}
}

func TestEmitAcceptsBoolFunctionResult(t *testing.T) {
	t.Parallel()
	program := lowerSource(t, "module main;\nfn positive(value: i32) -> bool { value > 0 }\nfn main() -> i32 { if positive(1) { 42 } else { 0 } }\n")
	unit, err := cbackend.Emit(program, "main::main", cbackend.Stage0Profile())
	if err != nil {
		t.Fatalf("Emit() error = %v", err)
	}
	if !slices.Equal(unit.Functions, []string{"main::main", "main::positive"}) {
		t.Fatalf("Emit() functions = %v", unit.Functions)
	}
}

func TestEmitScalarOptionAndResult(t *testing.T) {
	t.Parallel()
	program := lowerSource(t, `module main;
fn choose(present: bool) -> Option<i32> {
    if present { Option::Some(41) } else { Option::None }
}
fn increment(input: Result<i32, i32>) -> Result<i32, i32> {
    let value: i32 = input?;
    Result::Ok(value + 1)
}
fn main() -> i32 {
    let base: i32 = match choose(true) {
        Option::Some(value) => value,
        Option::None => 0,
    };
    match increment(Result::Ok(base)) {
        Result::Ok(value) => value,
        Result::Err(error) => -error,
    }
}
`)
	outcome, err := interpreter.Execute(program, "main::main")
	if err != nil || outcome.Kind != interpreter.OutcomeReturn || outcome.Value != 42 {
		t.Fatalf("interpreter = %+v, %v", outcome, err)
	}
	unit, err := cbackend.Emit(program, "main::main", cbackend.Stage0Profile())
	if err != nil {
		t.Fatalf("Emit() error = %v", err)
	}
	wantFeatures := []string{"checked-arithmetic", "scalar-call", "scalar-control-flow", "scalar-option", "scalar-result", "trap"}
	if !slices.Equal(unit.Features, wantFeatures) {
		t.Fatalf("Emit() features = %v, want %v", unit.Features, wantFeatures)
	}
	for _, required := range []string{
		"typedef struct { uint32_t tag; int32_t payload; } fy_option_i32;",
		"typedef struct { uint32_t tag; int32_t payload; } fy_result_i32_i32;",
		"static fy_option_i32",
		"static fy_result_i32_i32",
		"(fy_option_i32){ .tag = FY_OPTION_SOME, .payload = INT32_C(41) }",
		"(fy_result_i32_i32){ .tag = FY_RESULT_OK",
		".tag == FY_OPTION_SOME",
		".tag == FY_RESULT_OK",
		"fy_runtime_trap(FY_TRAP_UNREACHABLE)",
	} {
		if !strings.Contains(unit.Source, required) {
			t.Errorf("generated C is missing %q", required)
		}
	}
}

func TestEmitRejectsNestedScalarTag(t *testing.T) {
	t.Parallel()
	program := lowerSource(t, `module main;
fn main() -> i32 {
    let nested: Option<Option<i32>> = Option::Some(Option::Some(42));
    match nested {
        Option::Some(value) => match value {
            Option::Some(inner) => inner,
            Option::None => 0,
        },
        Option::None => 0,
    }
}
`)
	_, err := cbackend.Emit(program, "main::main", cbackend.Stage0Profile())
	var capability cbackend.CapabilityError
	if !errors.As(err, &capability) || !strings.Contains(capability.Feature, "Option construction type") {
		t.Fatalf("Emit() error = %v", err)
	}
}

func TestEmitCopyResultWithStructError(t *testing.T) {
	t.Parallel()
	program := lowerSource(t, `module main;
struct ConsoleError {
    status: u32;
    written: usize;
}
fn write(accept: bool) -> Result<usize, ConsoleError> {
    if accept {
        Result::Ok(42)
    } else {
        Result::Err(ConsoleError { status: 3, written: 7 })
    }
}
fn main() -> i32 {
    match write(false) {
        Result::Ok(_) => 0,
        Result::Err(error) => if error.status == 3 { 42 } else { 0 },
    }
}
`)
	outcome, err := interpreter.Execute(program, "main::main")
	if err != nil || outcome.Kind != interpreter.OutcomeReturn || outcome.Value != 42 {
		t.Fatalf("interpreter = %+v, %v", outcome, err)
	}
	unit, err := cbackend.Emit(program, "main::main", cbackend.Stage0Profile())
	if err != nil {
		t.Fatalf("Emit() error = %v", err)
	}
	wantFeatures := []string{"copy-result", "inert-struct", "scalar-call", "scalar-comparison", "scalar-control-flow", "trap"}
	if !slices.Equal(unit.Features, wantFeatures) {
		t.Fatalf("Emit() features = %v, want %v", unit.Features, wantFeatures)
	}
	for _, required := range []string{
		"_Static_assert(sizeof(fy_tagged_0) == UINT64_C(32)",
		"offsetof(fy_tagged_0, p_0) == UINT64_C(8)",
		"offsetof(fy_tagged_0, p_1) == UINT64_C(16)",
		"(fy_tagged_0){ .tag = FY_RESULT_OK, .p_0 =",
		"(fy_tagged_0){ .tag = FY_RESULT_ERR, .p_1 =",
		".p_1;",
	} {
		if !strings.Contains(unit.Source, required) {
			t.Errorf("generated C is missing %q", required)
		}
	}
}

func TestEmitRejectsResultWithEnumError(t *testing.T) {
	t.Parallel()
	program := lowerSource(t, `module main;
enum Failure { Denied; }
fn main() -> i32 {
    let result: Result<i32, Failure> = Result::Err(Failure::Denied);
    match result {
        Result::Ok(value) => value,
        Result::Err(_) => 42,
    }
}
`)
	_, err := cbackend.Emit(program, "main::main", cbackend.Stage0Profile())
	var capability cbackend.CapabilityError
	if !errors.As(err, &capability) || !strings.Contains(capability.Feature, "Result construction type") {
		t.Fatalf("Emit() error = %v", err)
	}
}

func TestEmitResourceBearingOption(t *testing.T) {
	t.Parallel()
	program := lowerSource(t, `module main;
resource Token {
    value: i32;
    drop fn(self: &mut Self) {}
}
fn main() -> i32 {
    let wrapped: Option<Token> = Option::Some(Token { value: 42 });
    match wrapped {
        Option::Some(token) => token.value,
        Option::None => 0,
    }
}
`)
	outcome, err := interpreter.Execute(program, "main::main")
	if err != nil || outcome.Kind != interpreter.OutcomeReturn || outcome.Value != 42 {
		t.Fatalf("interpreter = %+v, %v", outcome, err)
	}
	unit, err := cbackend.Emit(program, "main::main", cbackend.Stage0Profile())
	if err != nil {
		t.Fatalf("Emit() error = %v", err)
	}
	wantFeatures := []string{"flat-i32-resource", "resource-option", "scalar-control-flow", "trap"}
	if !slices.Equal(unit.Features, wantFeatures) {
		t.Fatalf("Emit() features = %v, want %v", unit.Features, wantFeatures)
	}
	for _, required := range []string{
		"} fy_tagged_0;",
		"_Static_assert(sizeof(fy_tagged_0) == UINT64_C(8)",
		"offsetof(fy_tagged_0, p_0) == UINT64_C(4)",
		"(fy_tagged_0){ .tag = FY_OPTION_SOME, .p_0 =",
		".p_0;",
	} {
		if !strings.Contains(unit.Source, required) {
			t.Errorf("generated C is missing %q", required)
		}
	}
}

func TestEmitRejectsNestedResourceTag(t *testing.T) {
	t.Parallel()
	program := lowerSource(t, `module main;
resource Token {
    value: i32;
    drop fn(self: &mut Self) {}
}
fn main() -> i32 {
    let wrapped: Option<Option<Token>> = Option::Some(Option::Some(Token { value: 42 }));
    42
}
`)
	_, err := cbackend.Emit(program, "main::main", cbackend.Stage0Profile())
	var capability cbackend.CapabilityError
	if !errors.As(err, &capability) || !strings.Contains(capability.Feature, "resource local type \"Option<Option<main::Token>>\"") {
		t.Fatalf("Emit() error = %v", err)
	}
}

func TestEmitResourceBearingResultLayout(t *testing.T) {
	t.Parallel()
	program := lowerSource(t, `module main;
resource Token {
    value: i32;
    drop fn(self: &mut Self) {}
}
fn main() -> i32 {
    let wrapped: Result<Token, i32> = Result::Ok(Token { value: 42 });
    match wrapped {
        Result::Ok(token) => token.value,
        Result::Err(error) => -error,
    }
}
`)
	unit, err := cbackend.Emit(program, "main::main", cbackend.Stage0Profile())
	if err != nil {
		t.Fatalf("Emit() error = %v", err)
	}
	for _, required := range []string{
		"_Static_assert(sizeof(fy_tagged_0) == UINT64_C(12)",
		"offsetof(fy_tagged_0, p_0) == UINT64_C(4)",
		"offsetof(fy_tagged_0, p_1) == UINT64_C(8)",
		"(fy_tagged_0){ .tag = FY_RESULT_OK, .p_0 =",
		".p_0;",
		".p_1;",
	} {
		if !strings.Contains(unit.Source, required) {
			t.Errorf("generated C is missing %q", required)
		}
	}
}

func TestEmitDropTraceProfileKeepsInstrumentationClosed(t *testing.T) {
	t.Parallel()
	program := lowerSource(t, `module main;
resource Token {
    value: i32;
    drop fn(self: &mut Self) {}
}
fn main() -> i32 {
    let direct: Token = Token { value: 1 };
    discard direct;
    let wrapped: Option<Token> = Option::Some(Token { value: 2 });
    discard wrapped;
    42
}
`)
	production, err := cbackend.Emit(program, "main::main", cbackend.Stage0Profile())
	if err != nil {
		t.Fatalf("production Emit() error = %v", err)
	}
	if production.Instrumentation != "" || len(production.DropSites) != 0 || strings.Contains(production.Source, "fy_runtime_trace_drop") {
		t.Fatalf("production unit contains conformance instrumentation: %+v", production)
	}
	instrumented, err := cbackend.Emit(program, "main::main", cbackend.Stage0DropTraceProfile())
	if err != nil {
		t.Fatalf("instrumented Emit() error = %v", err)
	}
	if instrumented.Instrumentation != cbackend.InstrumentationDropTraceV0 || len(instrumented.DropSites) != 2 {
		t.Fatalf("instrumented unit = %+v", instrumented)
	}
	for index, site := range instrumented.DropSites {
		if site.ID != uint32(index) || site.ResourceType != "main::Token" || site.Source != "main.fy" || site.StartByte >= site.EndByte {
			t.Errorf("drop site %d = %+v", index, site)
		}
	}
	for _, required := range []string{
		"void fy_runtime_trace_drop(uint32_t site_id);",
		"fy_runtime_trace_drop(UINT32_C(0));",
		"if (fy_l_",
		".tag == FY_OPTION_SOME) { fy_runtime_trace_drop(UINT32_C(1)); }",
	} {
		if !strings.Contains(instrumented.Source, required) {
			t.Errorf("instrumented C is missing %q", required)
		}
	}
}

func TestEmitRejectsMalformedScalarDestination(t *testing.T) {
	t.Parallel()
	program := lowerSource(t, "module main;\nfn main() -> i32 { let unused: i32 = 1; 42 }\n")
	mutated := mir.Clone(program)
	changed := false
	for functionIndex := range mutated.Functions {
		for blockIndex := range mutated.Functions[functionIndex].Blocks {
			for operationIndex := range mutated.Functions[functionIndex].Blocks[blockIndex].Operations {
				operation := &mutated.Functions[functionIndex].Blocks[blockIndex].Operations[operationIndex]
				if operation.Kind == mir.OperationAssign {
					operation.DestinationLocal = mir.NoLocal
					changed = true
					break
				}
			}
		}
	}
	if !changed {
		t.Fatal("fixture has no assignment to mutate")
	}
	if failures := mir.Verify(mutated); len(failures) != 0 {
		t.Fatalf("mutation must reach the C backend shape guard, verifier failures = %+v", failures)
	}
	_, err := cbackend.Emit(mutated, "main::main", cbackend.Stage0Profile())
	if err == nil || !strings.Contains(err.Error(), "no destination local") {
		t.Fatalf("Emit(malformed destination) error = %v", err)
	}
}

func TestEmitRejectsUnverifiedMIRAndUnknownProfile(t *testing.T) {
	t.Parallel()
	invalid := mir.Program{Functions: []mir.Function{{
		Name: "main::main", ResultType: hir.TypeI32, Entry: 0,
		Blocks: []mir.Block{{ID: 0, Terminator: mir.Terminator{UnwindSuccessor: mir.NoBlock}}},
	}}}
	if _, err := cbackend.Emit(invalid, "main::main", cbackend.Stage0Profile()); err == nil || !strings.Contains(err.Error(), "verified MIR") {
		t.Fatalf("Emit(unverified) error = %v", err)
	}
	program := lowerSource(t, "module main;\nfn main() -> i32 { 0 }\n")
	if _, err := cbackend.Emit(program, "main::main", cbackend.Profile{Target: "host", Dialect: cbackend.DialectC11}); err == nil || !strings.Contains(err.Error(), "unsupported C emission profile") {
		t.Fatalf("Emit(profile) error = %v", err)
	}
	invalidInstrumentation := cbackend.Stage0Profile()
	invalidInstrumentation.Instrumentation = "unbounded-observer"
	if _, err := cbackend.Emit(program, "main::main", invalidInstrumentation); err == nil || !strings.Contains(err.Error(), "unsupported C emission profile") {
		t.Fatalf("Emit(instrumentation) error = %v", err)
	}
}

func TestEmitDeduplicatesCompilerOwnedUTF8StringStorage(t *testing.T) {
	t.Parallel()
	program := lowerSource(t, `module main;
fn main() -> i32 {
    let first: &str = "λ";
    let second: &str = "λ";
    let empty: &str = "";
    42
}
`)
	unit, err := cbackend.Emit(program, "main::main", cbackend.Stage0Profile())
	if err != nil {
		t.Fatalf("Emit() error = %v", err)
	}
	if !slices.Equal(unit.Features, []string{"static-utf8-string"}) {
		t.Fatalf("features = %v", unit.Features)
	}
	if strings.Count(unit.Source, "static const uint8_t fy_static_str_") != 2 {
		t.Fatalf("generated C did not deduplicate UTF-8 string storage or preserve the empty sentinel:\n%s", unit.Source)
	}
	for _, required := range []string{
		"typedef struct fy_str",
		"const uint8_t *data;",
		"uintptr_t length;",
		"[2] = { UINT8_C(206), UINT8_C(187) };",
		"[1] = { UINT8_C(0) };",
		"fy_str fy_l_",
		"(void)&fy_l_",
		"((uintptr_t)UINT64_C(2))",
		"((uintptr_t)UINT64_C(0))",
	} {
		if !strings.Contains(unit.Source, required) {
			t.Errorf("generated C is missing %q:\n%s", required, unit.Source)
		}
	}
	second, err := cbackend.Emit(program, "main::main", cbackend.Stage0Profile())
	if err != nil || second.Source != unit.Source {
		t.Fatalf("second Emit() differs: error=%v", err)
	}
}

func TestEmitUTF8StringLocalDataAddressForForeignCall(t *testing.T) {
	t.Parallel()
	program := lowerSource(t, `module main;
@link_name("fairyc_fixture_is_fairy_text")
unsafe extern C fn is_fairy_text(value: *u8) -> u32;
fn main() -> i32 {
    let text: &str = "fairy";
    if unsafe { is_fairy_text(text as *u8) } == 1 { 42 } else { 1 }
}
`)
	unit, err := cbackend.Emit(program, "main::main", cbackend.Stage0Profile())
	if err != nil {
		t.Fatalf("Emit() error = %v", err)
	}
	for _, feature := range []string{"foreign-c-call", "static-utf8-string", "str-data-address"} {
		if !slices.Contains(unit.Features, feature) {
			t.Errorf("features %v are missing %q", unit.Features, feature)
		}
	}
	if !slices.Equal(unit.ForeignSymbols, []string{"fairyc_fixture_is_fairy_text"}) {
		t.Fatalf("foreign symbols = %v", unit.ForeignSymbols)
	}
	for _, required := range []string{
		"extern uint32_t fairyc_fixture_is_fairy_text(const uint8_t * fy_a_0);",
		"static const uint8_t fy_static_str_",
		".length = ((uintptr_t)UINT64_C(5))",
		".data;",
		"fairyc_fixture_is_fairy_text(fy_l_",
	} {
		if !strings.Contains(unit.Source, required) {
			t.Errorf("generated C is missing %q:\n%s", required, unit.Source)
		}
	}
}

func TestEmitAlwaysCheckedAssert(t *testing.T) {
	t.Parallel()
	program := lowerSource(t, "module main;\nfn main() -> i32 { let ready: bool = true; assert(ready); 42 }\n")
	unit, err := cbackend.Emit(program, "main::main", cbackend.Stage0Profile())
	if err != nil {
		t.Fatalf("Emit() error = %v", err)
	}
	for _, feature := range []string{"assert", "trap"} {
		if !slices.Contains(unit.Features, feature) {
			t.Errorf("features %v are missing %q", unit.Features, feature)
		}
	}
	for _, required := range []string{
		"FY_TRAP_ASSERTION_FAILED",
		"== INT32_C(0)) { fy_pending_trap = FY_TRAP_ASSERTION_FAILED;",
	} {
		if !strings.Contains(unit.Source, required) {
			t.Errorf("generated C is missing %q:\n%s", required, unit.Source)
		}
	}
}

func lowerSource(t *testing.T, contents string) mir.Program {
	t.Helper()
	return lowerSources(t, map[string]string{"main.fy": contents})
}

func lowerSources(t *testing.T, contents map[string]string) mir.Program {
	t.Helper()
	paths := make([]string, 0, len(contents))
	for path := range contents {
		paths = append(paths, path)
	}
	slices.Sort(paths)
	files := make([]semantic.File, 0, len(paths))
	for _, path := range paths {
		file := source.NewFile(path, []byte(contents[path]))
		parsed, err := frontend.AnalyzeSyntax(file, source.DefaultLimits())
		if err != nil {
			t.Fatalf("AnalyzeSyntax(%s) error = %v", path, err)
		}
		if len(parsed.Tree.Diagnostics) != 0 || parsed.AST == nil {
			t.Fatalf("AnalyzeSyntax(%s) diagnostics = %+v", path, parsed.Tree.Diagnostics)
		}
		files = append(files, semantic.File{Path: path, Module: parsed.AST})
	}
	semanticResult := semantic.Analyze(files)
	if len(semanticResult.Diagnostics) != 0 || len(semanticResult.Unsupported) != 0 {
		t.Fatalf("Analyze() diagnostics = %+v, unsupported = %+v", semanticResult.Diagnostics, semanticResult.Unsupported)
	}
	program, err := mir.Lower(semanticResult.HIR)
	if err != nil {
		t.Fatalf("Lower() error = %v", err)
	}
	return program
}
