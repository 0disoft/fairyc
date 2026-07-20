package cbackendnative

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/cbackend"
	"github.com/0disoft/fairyc/internal/cbackendfixture"
	"github.com/0disoft/fairyc/internal/cdriver"
	"github.com/0disoft/fairyc/internal/cli"
	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/interpreter"
	"github.com/0disoft/fairyc/internal/layoutfixture"
	"github.com/0disoft/fairyc/internal/target"
)

func TestBuildSmokeSourceReachesCEmission(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "main.fy"), []byte(buildSmokeSource), 0o600); err != nil {
		t.Fatal(err)
	}
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := cli.Run([]string{"experimental", "emit-c", "main.fy"}, root, &stdout, &stderr)
	if code != cli.ExitSuccess || stderr.Len() != 0 {
		t.Fatalf("build smoke C emission = code %d, stderr %q", code, stderr.String())
	}
	for _, required := range []string{"fy_resource_0", "fy_tagged_0", "fy_tagged_1", "FY_OPTION_SOME", "FY_RESULT_OK"} {
		if !strings.Contains(stdout.String(), required) {
			t.Errorf("build smoke C emission is missing %q", required)
		}
	}
	if strings.Contains(stdout.String(), "fy_runtime_trace_drop") {
		t.Fatal("product emit-c output contains native conformance instrumentation")
	}
}

func TestForeignPointerCallSmokeSourceReachesCEmission(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "main.fy"), []byte(foreignPointerCallSmokeSource), 0o600); err != nil {
		t.Fatal(err)
	}
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := cli.Run([]string{"experimental", "emit-c", "main.fy"}, root, &stdout, &stderr)
	if code != cli.ExitSuccess || stderr.Len() != 0 {
		t.Fatalf("foreign pointer call C emission = code %d, stderr %q", code, stderr.String())
	}
	for _, required := range []string{
		"extern uint32_t fairyc_fixture_pointer_scalars(const uint8_t * fy_a_0, uint32_t * fy_a_1, uintptr_t fy_a_2, uint8_t fy_a_3);",
		"fairyc_fixture_pointer_scalars(((void *)0), ((void *)0), ((uintptr_t)UINT64_C(0)), UINT8_C(7))",
	} {
		if !strings.Contains(stdout.String(), required) {
			t.Errorf("foreign pointer call C emission is missing %q", required)
		}
	}
}

func TestForeignCStructResultCallSmokeSourceReachesCEmission(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "main.fy"), []byte(foreignCStructResultCallSmokeSource), 0o600); err != nil {
		t.Fatal(err)
	}
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := cli.Run([]string{"experimental", "emit-c", "main.fy"}, root, &stdout, &stderr)
	if code != cli.ExitSuccess || stderr.Len() != 0 {
		t.Fatalf("foreign C struct-result call C emission = code %d, stderr %q", code, stderr.String())
	}
	for _, required := range foreignCStructResultCallExpectedCFragments {
		if !strings.Contains(stdout.String(), required) {
			t.Errorf("foreign C struct-result call C emission is missing %q", required)
		}
	}
}

func TestForeignCIntCallSmokeSourceReachesCEmission(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "main.fy"), []byte(foreignCIntCallSmokeSource), 0o600); err != nil {
		t.Fatal(err)
	}
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := cli.Run([]string{"experimental", "emit-c", "main.fy"}, root, &stdout, &stderr)
	if code != cli.ExitSuccess || stderr.Len() != 0 {
		t.Fatalf("foreign c::int call C emission = code %d, stderr %q", code, stderr.String())
	}
	for _, required := range []string{
		"_Static_assert(sizeof(int) == UINT64_C(4), \"fairyc target c.int size\");",
		"extern int fairyc_fixture_make_c_int(void);",
		"extern int32_t fairyc_fixture_consume_c_int(int fy_a_0);",
		"fairyc_fixture_consume_c_int(fy_l_",
	} {
		if !strings.Contains(stdout.String(), required) {
			t.Errorf("foreign c::int call C emission is missing %q", required)
		}
	}
}

func TestForeignCSizeTCallSmokeSourceReachesCEmission(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "main.fy"), []byte(foreignCSizeTCallSmokeSource), 0o600); err != nil {
		t.Fatal(err)
	}
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := cli.Run([]string{"experimental", "emit-c", "main.fy"}, root, &stdout, &stderr)
	if code != cli.ExitSuccess || stderr.Len() != 0 {
		t.Fatalf("foreign c::size_t call C emission = code %d, stderr %q", code, stderr.String())
	}
	for _, required := range []string{
		"#include <stddef.h>",
		"_Static_assert(sizeof(size_t) == UINT64_C(8), \"fairyc target c.size_t size\");",
		"extern size_t fairyc_fixture_make_c_size_t(void);",
		"extern int32_t fairyc_fixture_consume_c_size_t(size_t fy_a_0);",
	} {
		if !strings.Contains(stdout.String(), required) {
			t.Errorf("foreign c::size_t call C emission is missing %q", required)
		}
	}
}

func TestForeignCLongCallSmokeSourceReachesCEmission(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "main.fy"), []byte(foreignCLongCallSmokeSource), 0o600); err != nil {
		t.Fatal(err)
	}
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := cli.Run([]string{"experimental", "emit-c", "main.fy"}, root, &stdout, &stderr)
	if code != cli.ExitSuccess || stderr.Len() != 0 {
		t.Fatalf("foreign c::long call C emission = code %d, stderr %q", code, stderr.String())
	}
	for _, required := range []string{
		"_Static_assert(sizeof(long) == UINT64_C(8), \"fairyc target c.long size\");",
		"_Static_assert(((long)-1) < (long)0, \"fairyc target c.long signed\");",
		"extern long fairyc_fixture_make_c_long(void);",
		"extern int32_t fairyc_fixture_consume_c_long(long fy_a_0);",
	} {
		if !strings.Contains(stdout.String(), required) {
			t.Errorf("foreign c::long call C emission is missing %q", required)
		}
	}
}

func TestForeignCScalarFamilyCallSmokeSourceReachesCEmission(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "main.fy"), []byte(foreignCScalarFamilyCallSmokeSource), 0o600); err != nil {
		t.Fatal(err)
	}
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := cli.Run([]string{"experimental", "emit-c", "main.fy"}, root, &stdout, &stderr)
	if code != cli.ExitSuccess || stderr.Len() != 0 {
		t.Fatalf("foreign C scalar family C emission = code %d, stderr %q", code, stderr.String())
	}
	for _, required := range foreignCScalarFamilyExpectedCFragments {
		if !strings.Contains(stdout.String(), required) {
			t.Errorf("foreign C scalar family C emission is missing %q", required)
		}
	}
}

func TestForeignCSSizeTCallSmokeSourceReachesCEmission(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "main.fy"), []byte(foreignCSSizeTCallSmokeSource), 0o600); err != nil {
		t.Fatal(err)
	}
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := cli.Run([]string{"experimental", "emit-c", "main.fy"}, root, &stdout, &stderr)
	if code != cli.ExitSuccess || stderr.Len() != 0 {
		t.Fatalf("foreign c::ssize_t call C emission = code %d, stderr %q", code, stderr.String())
	}
	for _, required := range []string{
		"#define _POSIX_C_SOURCE 200809L",
		"#include <sys/types.h>",
		"_Static_assert(sizeof(ssize_t) == UINT64_C(8), \"fairyc target c.ssize_t size\");",
		"extern ssize_t fairyc_fixture_make_c_ssize_t(void);",
		"extern int32_t fairyc_fixture_consume_c_ssize_t(ssize_t fy_a_0);",
	} {
		if !strings.Contains(stdout.String(), required) {
			t.Errorf("foreign c::ssize_t call C emission is missing %q", required)
		}
	}
}

func TestTextValidationSourceCallSmokeReachesCEmission(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "main.fy"), []byte(textValidationSourceCallSmokeSource), 0o600); err != nil {
		t.Fatal(err)
	}
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := cli.Run([]string{"experimental", "emit-c", "main.fy"}, root, &stdout, &stderr)
	if code != cli.ExitSuccess || stderr.Len() != 0 {
		t.Fatalf("text validation source call C emission = code %d, stderr %q", code, stderr.String())
	}
	for _, required := range []string{
		"extern uint32_t fy_std_text_utf8_valid_v0(const uint8_t * fy_a_0, uintptr_t fy_a_1);",
		"fy_std_text_utf8_valid_v0(((void *)0), ((uintptr_t)UINT64_C(0)))",
	} {
		if !strings.Contains(stdout.String(), required) {
			t.Errorf("text validation source call C emission is missing %q", required)
		}
	}
}

func TestTextViewOutputSourceCallSmokeReachesCEmission(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "main.fy"), []byte(textViewOutputSourceCallSmokeSource), 0o600); err != nil {
		t.Fatal(err)
	}
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := cli.Run([]string{"experimental", "emit-c", "main.fy"}, root, &stdout, &stderr)
	if code != cli.ExitSuccess || stderr.Len() != 0 {
		t.Fatalf("text view output source call C emission = code %d, stderr %q", code, stderr.String())
	}
	for _, required := range []string{
		"struct fy_std_text_view_v0 {",
		"const uint8_t * data;",
		"uintptr_t length;",
		"extern uint32_t fy_std_text_validate_utf8_v0(const uint8_t * fy_a_0, uintptr_t fy_a_1, struct fy_std_text_view_v0 * fy_a_2);",
		" = &fy_l_",
		"fy_std_text_validate_utf8_v0(((void *)0), ((uintptr_t)UINT64_C(0)), fy_l_",
		".length",
	} {
		if !strings.Contains(stdout.String(), required) {
			t.Errorf("text view output source call C emission is missing %q", required)
		}
	}
}

func TestSystemHeapSourceCallSmokeReachesCEmission(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "main.fy"), []byte(systemHeapSourceCallSmokeSource), 0o600); err != nil {
		t.Fatal(err)
	}
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := cli.Run([]string{"experimental", "emit-c", "main.fy"}, root, &stdout, &stderr)
	if code != cli.ExitSuccess || stderr.Len() != 0 {
		t.Fatalf("SystemHeap source call C emission = code %d, stderr %q", code, stderr.String())
	}
	for _, required := range systemHeapSourceCallExpectedCFragments {
		if !strings.Contains(stdout.String(), required) {
			t.Errorf("SystemHeap source call C emission is missing %q", required)
		}
	}
}

func TestBufferSourceCallSmokeReachesCEmission(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "main.fy"), []byte(bufferSourceCallSmokeSource), 0o600); err != nil {
		t.Fatal(err)
	}
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := cli.Run([]string{"experimental", "emit-c", "main.fy"}, root, &stdout, &stderr)
	if code != cli.ExitSuccess || stderr.Len() != 0 {
		t.Fatalf("Buffer source call C emission = code %d, stderr %q", code, stderr.String())
	}
	for _, required := range bufferSourceCallExpectedCFragments {
		if !strings.Contains(stdout.String(), required) {
			t.Errorf("Buffer source call C emission is missing %q", required)
		}
	}
}

func TestArenaSourceCallSmokeReachesCEmission(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "main.fy"), []byte(arenaSourceCallSmokeSource), 0o600); err != nil {
		t.Fatal(err)
	}
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := cli.Run([]string{"experimental", "emit-c", "main.fy"}, root, &stdout, &stderr)
	if code != cli.ExitSuccess || stderr.Len() != 0 {
		t.Fatalf("Arena source call C emission = code %d, stderr %q", code, stderr.String())
	}
	for _, required := range arenaSourceCallExpectedCFragments {
		if !strings.Contains(stdout.String(), required) {
			t.Errorf("Arena source call C emission is missing %q", required)
		}
	}
}

func TestCStrValidationSourceCallSmokeReachesCEmission(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "main.fy"), []byte(cStrValidationSourceCallSmokeSource), 0o600); err != nil {
		t.Fatal(err)
	}
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := cli.Run([]string{"experimental", "emit-c", "main.fy"}, root, &stdout, &stderr)
	if code != cli.ExitSuccess || stderr.Len() != 0 {
		t.Fatalf("CStr validation source call C emission = code %d, stderr %q", code, stderr.String())
	}
	for _, required := range cStrValidationSourceCallExpectedCFragments {
		if !strings.Contains(stdout.String(), required) {
			t.Errorf("CStr validation source call C emission is missing %q", required)
		}
	}
}

func TestOwnedTextSourceCallSmokeReachesCEmission(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "main.fy"), []byte(ownedTextSourceCallSmokeSource), 0o600); err != nil {
		t.Fatal(err)
	}
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := cli.Run([]string{"experimental", "emit-c", "main.fy"}, root, &stdout, &stderr)
	if code != cli.ExitSuccess || stderr.Len() != 0 {
		t.Fatalf("owned-text source call C emission = code %d, stderr %q", code, stderr.String())
	}
	for _, required := range ownedTextSourceCallExpectedCFragments {
		if !strings.Contains(stdout.String(), required) {
			t.Errorf("owned-text source call C emission is missing %q", required)
		}
	}
}

func TestLinuxFileSourceCallSmokeReachesCEmission(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "main.fy"), []byte(linuxFileSourceCallSmokeSource), 0o600); err != nil {
		t.Fatal(err)
	}
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := cli.Run([]string{"experimental", "emit-c", "main.fy"}, root, &stdout, &stderr)
	if code != cli.ExitSuccess || stderr.Len() != 0 {
		t.Fatalf("Linux File source call C emission = code %d, stderr %q", code, stderr.String())
	}
	for _, required := range linuxFileSourceCallExpectedCFragments {
		if !strings.Contains(stdout.String(), required) {
			t.Errorf("Linux File source call C emission is missing %q", required)
		}
	}
}

func TestCStringLiteralSourceCallSmokeReachesCEmission(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "main.fy"), []byte(cStringLiteralSourceCallSmokeSource), 0o600); err != nil {
		t.Fatal(err)
	}
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := cli.Run([]string{"experimental", "emit-c", "main.fy"}, root, &stdout, &stderr)
	if code != cli.ExitSuccess || stderr.Len() != 0 {
		t.Fatalf("C string literal source call C emission = code %d, stderr %q", code, stderr.String())
	}
	for _, required := range cStringLiteralSourceCallExpectedCFragments {
		if !strings.Contains(stdout.String(), required) {
			t.Errorf("C string literal source call C emission is missing %q", required)
		}
	}
}

func TestByteStringLiteralSourceCallSmokeReachesCEmission(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "main.fy"), []byte(byteStringLiteralSourceCallSmokeSource), 0o600); err != nil {
		t.Fatal(err)
	}
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := cli.Run([]string{"experimental", "emit-c", "main.fy"}, root, &stdout, &stderr)
	if code != cli.ExitSuccess || stderr.Len() != 0 {
		t.Fatalf("byte-string literal source call C emission = code %d, stderr %q", code, stderr.String())
	}
	for _, required := range byteStringLiteralSourceCallExpectedCFragments {
		if !strings.Contains(stdout.String(), required) {
			t.Errorf("byte-string literal source call C emission is missing %q", required)
		}
	}
}

func TestUTF8StringLiteralSourceCallSmokeReachesCEmission(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "main.fy"), []byte(utf8StringLiteralSourceCallSmokeSource), 0o600); err != nil {
		t.Fatal(err)
	}
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := cli.Run([]string{"experimental", "emit-c", "main.fy"}, root, &stdout, &stderr)
	if code != cli.ExitSuccess || stderr.Len() != 0 {
		t.Fatalf("UTF-8 string literal source call C emission = code %d, stderr %q", code, stderr.String())
	}
	for _, required := range utf8StringLiteralSourceCallExpectedCFragments {
		if !strings.Contains(stdout.String(), required) {
			t.Errorf("UTF-8 string literal source call C emission is missing %q", required)
		}
	}
}

func TestNativeReportSchemaTracksNativeEvidenceShape(t *testing.T) {
	t.Parallel()
	if reportSchemaVersion != "fairyc.native-c-report/v31" {
		t.Fatalf("report schema = %q", reportSchemaVersion)
	}
	encoded, err := json.Marshal(Report{
		SchemaVersion: reportSchemaVersion, Executions: 80, DropTraceExecutions: 80,
		LayoutCases: []string{"pass.array-i32-three", "pass.pointer"}, LayoutProbeExecutions: 4,
		Sanitizer: cdriver.AddressUndefinedSanitizerMode, SanitizerExecutions: 40,
		ABICases:      []string{"pass.aggregate-rollback", "pass.void-return"},
		ABIDirections: []string{"clang-caller-gcc-callee", "gcc-caller-clang-callee"}, ABIProbeExecutions: 56,
		SystemHeapExecutions:                4,
		SystemHeapSourceCallExecutions:      4,
		BufferExecutions:                    4,
		BufferSourceCallExecutions:          4,
		ArenaExecutions:                     4,
		ArenaSourceCallExecutions:           4,
		TextViewExecutions:                  4,
		OwnedTextExecutions:                 4,
		OwnedTextSourceCallExecutions:       4,
		LinuxFileExecutions:                 4,
		LinuxFileSourceCallExecutions:       4,
		ForeignCallExecutions:               4,
		ForeignCIntCallExecutions:           4,
		ForeignCLongCallExecutions:          4,
		ForeignCScalarFamilyCallExecutions:  4,
		ForeignCSizeTCallExecutions:         4,
		ForeignCSSizeTCallExecutions:        4,
		ForeignPointerCallExecutions:        4,
		ForeignCStructResultCallExecutions:  4,
		TextValidationSourceCallExecutions:  4,
		TextViewOutputSourceCallExecutions:  4,
		CStrValidationSourceCallExecutions:  4,
		ByteStringSourceCallExecutions:      4,
		UTF8StringSourceCallExecutions:      4,
		CStringLiteralSourceCallExecutions:  4,
		BuildSmokeExecutions:                4,
		StandardLibraryBuildSmokeExecutions: 4,
		ByteStringBuildSmokeExecutions:      4,
		UTF8StringBuildSmokeExecutions:      4,
		CStringLiteralBuildSmokeExecutions:  4,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(encoded), `"schemaVersion":"fairyc.native-c-report/v31"`) ||
		!strings.Contains(string(encoded), `"dropTraceExecutions":80`) ||
		!strings.Contains(string(encoded), `"layoutCases":["pass.array-i32-three","pass.pointer"]`) ||
		!strings.Contains(string(encoded), `"layoutProbeExecutions":4`) ||
		!strings.Contains(string(encoded), `"sanitizer":"address-undefined-o1"`) ||
		!strings.Contains(string(encoded), `"sanitizerExecutions":40`) ||
		!strings.Contains(string(encoded), `"abiCases":["pass.aggregate-rollback","pass.void-return"]`) ||
		!strings.Contains(string(encoded), `"abiDirections":["clang-caller-gcc-callee","gcc-caller-clang-callee"]`) ||
		!strings.Contains(string(encoded), `"abiProbeExecutions":56`) ||
		!strings.Contains(string(encoded), `"systemHeapExecutions":4`) ||
		!strings.Contains(string(encoded), `"systemHeapSourceCallExecutions":4`) ||
		!strings.Contains(string(encoded), `"bufferExecutions":4`) ||
		!strings.Contains(string(encoded), `"bufferSourceCallExecutions":4`) ||
		!strings.Contains(string(encoded), `"arenaExecutions":4`) ||
		!strings.Contains(string(encoded), `"arenaSourceCallExecutions":4`) ||
		!strings.Contains(string(encoded), `"textViewExecutions":4`) ||
		!strings.Contains(string(encoded), `"ownedTextExecutions":4`) ||
		!strings.Contains(string(encoded), `"ownedTextSourceCallExecutions":4`) ||
		!strings.Contains(string(encoded), `"linuxFileExecutions":4`) ||
		!strings.Contains(string(encoded), `"linuxFileSourceCallExecutions":4`) ||
		!strings.Contains(string(encoded), `"foreignCallExecutions":4`) ||
		!strings.Contains(string(encoded), `"foreignCIntCallExecutions":4`) ||
		!strings.Contains(string(encoded), `"foreignCLongCallExecutions":4`) ||
		!strings.Contains(string(encoded), `"foreignCScalarFamilyCallExecutions":4`) ||
		!strings.Contains(string(encoded), `"foreignCSizeTCallExecutions":4`) ||
		!strings.Contains(string(encoded), `"foreignCSSizeTCallExecutions":4`) ||
		!strings.Contains(string(encoded), `"foreignPointerCallExecutions":4`) ||
		!strings.Contains(string(encoded), `"foreignCStructResultCallExecutions":4`) ||
		!strings.Contains(string(encoded), `"textValidationSourceCallExecutions":4`) ||
		!strings.Contains(string(encoded), `"textViewOutputSourceCallExecutions":4`) ||
		!strings.Contains(string(encoded), `"cStrValidationSourceCallExecutions":4`) ||
		!strings.Contains(string(encoded), `"byteStringLiteralSourceCallExecutions":4`) ||
		!strings.Contains(string(encoded), `"utf8StringLiteralSourceCallExecutions":4`) ||
		!strings.Contains(string(encoded), `"cStringLiteralSourceCallExecutions":4`) ||
		!strings.Contains(string(encoded), `"buildSmokeExecutions":4`) ||
		!strings.Contains(string(encoded), `"standardLibraryBuildSmokeExecutions":4`) ||
		!strings.Contains(string(encoded), `"byteStringLiteralBuildSmokeExecutions":4`) ||
		!strings.Contains(string(encoded), `"utf8StringLiteralBuildSmokeExecutions":4`) ||
		!strings.Contains(string(encoded), `"cStringLiteralBuildSmokeExecutions":4`) {
		t.Fatalf("report JSON = %s", encoded)
	}
}

func TestSystemHeapProbeRunnerRejectsEmptySourceBeforeToolchainUse(t *testing.T) {
	t.Parallel()
	toolchains := []cdriver.Toolchain{{Name: "clang"}, {Name: "gcc"}}
	if _, err := runStandardLibraryProbe(t.TempDir(), toolchains, "SystemHeap", "system-heap", "", ""); err == nil || !strings.Contains(err.Error(), "must be non-empty") {
		t.Fatalf("runStandardLibraryProbe() error = %v", err)
	}
}

func TestLinuxFileSourceCallProbeRejectsEmptySourceBeforeToolchainUse(t *testing.T) {
	t.Parallel()
	toolchains := []cdriver.Toolchain{{Name: "clang"}, {Name: "gcc"}}
	if _, err := runLinuxFileSourceCallProbe(t.TempDir(), toolchains, ""); err == nil || !strings.Contains(err.Error(), "requires a non-empty implementation") {
		t.Fatalf("runLinuxFileSourceCallProbe() error = %v", err)
	}
}

func TestSanitizerEnvironmentOverridesAmbientConfiguration(t *testing.T) {
	t.Parallel()
	environment := sanitizerEnvironment([]string{
		"PATH=/bin", "ASAN_OPTIONS=detect_leaks=0", "LSAN_OPTIONS=exitcode=0", "UBSAN_OPTIONS=halt_on_error=0", "HOME=/tmp/home",
	})
	if len(environment) != 5 || environment[0] != "PATH=/bin" || environment[1] != "HOME=/tmp/home" {
		t.Fatalf("sanitizer environment = %v", environment)
	}
	joined := strings.Join(environment, "\n")
	for _, required := range []string{
		"ASAN_OPTIONS=abort_on_error=1:color=never:detect_leaks=1:halt_on_error=1:symbolize=0",
		"LSAN_OPTIONS=exitcode=23:report_objects=0",
		"UBSAN_OPTIONS=color=never:halt_on_error=1:print_stacktrace=0",
	} {
		if strings.Count(joined, required) != 1 {
			t.Errorf("sanitizer environment is missing unique %q: %v", required, environment)
		}
	}
}

func TestGenerateLayoutProbeFromRepositoryFixtures(t *testing.T) {
	t.Parallel()
	manifest := filepath.Join("..", "..", "spec", "fixtures", "layout", "manifest-v0.json")
	targetID, cases, err := layoutfixture.NativeProbeCases(manifest)
	if err != nil {
		t.Fatal(err)
	}
	unit, err := generateLayoutProbe(targetID, cases)
	if err != nil {
		t.Fatal(err)
	}
	repeated, err := generateLayoutProbe(targetID, cases)
	if err != nil || repeated.Source != unit.Source || !slices.Equal(repeated.Cases, unit.Cases) {
		t.Fatalf("repeated layout probe drifted: %v", err)
	}
	if unit.Target != target.X8664LinuxGNU || len(unit.Cases) != 11 || !strings.HasSuffix(unit.Source, "int main(void) { return 0; }\n") {
		t.Fatalf("layout probe unit = %+v", unit)
	}
	for _, required := range []string{
		"typedef int32_t",
		"typedef long",
		"typedef size_t",
		"typedef ssize_t",
		"typedef void *",
		"typedef void (*",
		"_Static_assert(sizeof(",
		"_Static_assert(_Alignof(",
		"_Static_assert(offsetof(",
	} {
		if !strings.Contains(unit.Source, required) {
			t.Errorf("layout probe is missing %q", required)
		}
	}
	for _, probeCase := range unit.Cases {
		if strings.Contains(unit.Source, probeCase) {
			t.Errorf("layout probe source leaks fixture case ID %q", probeCase)
		}
	}
}

func TestExpectedExitCode(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		fixture cbackendfixture.EmittedCase
		want    int
		fails   bool
	}{
		{name: "return", fixture: cbackendfixture.EmittedCase{ID: "return", Outcome: "return", Value: 42}, want: 42},
		{name: "overflow", fixture: cbackendfixture.EmittedCase{ID: "trap", Outcome: "trap", TrapCategory: "integer-overflow"}, want: 194},
		{name: "unknown trap", fixture: cbackendfixture.EmittedCase{ID: "trap", Outcome: "trap", TrapCategory: "unknown"}, fails: true},
		{name: "negative return", fixture: cbackendfixture.EmittedCase{ID: "return", Outcome: "return", Value: -1}, fails: true},
		{name: "reserved return", fixture: cbackendfixture.EmittedCase{ID: "return", Outcome: "return", Value: trapExitBase}, fails: true},
	}
	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			got, err := expectedExitCode(test.fixture)
			if test.fails {
				if err == nil {
					t.Fatalf("expectedExitCode() = %d, nil", got)
				}
				return
			}
			if err != nil || got != test.want {
				t.Fatalf("expectedExitCode() = %d, %v; want %d, nil", got, err, test.want)
			}
		})
	}
}

func TestDropTraceRecordsUseSourceLinkedSiteIDs(t *testing.T) {
	t.Parallel()
	span := hir.Span{Source: "main.fy", StartByte: 10, EndByte: 15}
	fixture := cbackendfixture.EmittedCase{
		ID: "drop",
		DropTrace: []interpreter.DropEvent{
			{ResourceType: "main::Token", Binding: span},
			{ResourceType: "main::Token", Binding: span},
		},
		TraceUnit: cbackend.Unit{DropSites: []cbackend.DropSite{{
			ID: 7, ResourceType: "main::Token", Source: "main.fy", StartByte: 10, EndByte: 15,
		}}},
	}
	ids, err := expectedDropSiteIDs(fixture)
	if err != nil || !slices.Equal(ids, []uint32{7, 7}) {
		t.Fatalf("expectedDropSiteIDs() = %v, %v", ids, err)
	}
	decoded, err := decodeDropTrace([]byte{7, 0, 0, 0, 7, 0, 0, 0})
	if err != nil || !slices.Equal(decoded, ids) {
		t.Fatalf("decodeDropTrace() = %v, %v", decoded, err)
	}
	fixture.DropTrace[0].Binding.StartByte++
	if _, err := expectedDropSiteIDs(fixture); err == nil {
		t.Fatal("expectedDropSiteIDs() accepted a missing source-linked site")
	}
	if _, err := decodeDropTrace([]byte{1, 2, 3}); err == nil {
		t.Fatal("decodeDropTrace() accepted a partial record")
	}
}

func TestBoundedBufferRetainsPrefix(t *testing.T) {
	t.Parallel()
	buffer := boundedBuffer{limit: 4}
	if written, err := buffer.Write([]byte("abcdef")); err != nil || written != 6 {
		t.Fatalf("Write() = %d, %v", written, err)
	}
	if buffer.String() != "abcd" || !buffer.truncated {
		t.Fatalf("buffer = %q, truncated=%t", buffer.String(), buffer.truncated)
	}
}

func TestRunRejectsNonLinuxHost(t *testing.T) {
	if runtime.GOOS == "linux" && runtime.GOARCH == "amd64" {
		t.Skip("native host is eligible; actual compiler execution belongs to the configured native command")
	}
	_, err := Run("unused.json")
	if err == nil || !strings.Contains(err.Error(), "requires linux/amd64") {
		t.Fatalf("Run() error = %v", err)
	}
}
