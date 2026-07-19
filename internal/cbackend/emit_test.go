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

func TestEmitRejectsNonI32ResourceField(t *testing.T) {
	t.Parallel()
	program := lowerSource(t, "module main;\nresource Flag { value: bool; drop fn(self: &mut Self) {} }\nfn main() -> i32 { 0 }\n")
	_, err := cbackend.Emit(program, "main::main", cbackend.Stage0Profile())
	var capability cbackend.CapabilityError
	if !errors.As(err, &capability) || !strings.Contains(capability.Feature, "field \"value\" of type \"bool\"") || capability.Function != "" {
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
