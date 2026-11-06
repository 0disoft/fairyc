package cbackend_test

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/0disoft/fairyc/internal/cbackend"
	"github.com/0disoft/fairyc/internal/cdriver"
	"github.com/0disoft/fairyc/internal/cruntime"
	"github.com/0disoft/fairyc/internal/target"
)

func TestPartitionedBuildIdentityAndExecution(t *testing.T) {
	program := lowerSource(t, `module main;
unsafe extern C fn i32 same_literal();
fn ref u8[] first() { return b"shared"; }
fn ref u8[] second() { return b"shared"; }
fn i32 shifted(i32 value) { return value >> 1; }
fn Result<i32,i32> choose(bool ok) { return if(ok) { .ok(42) } else { .err(1) }; }
fn i32 main() {
    assert(first()[0] == second()[0]);
    assert(unsafe { same_literal() } == 1);
    let i32 value = switch(choose(true)) { case .ok(n): { n } case .err(e): { -e } };
    assert(shifted(-4) == -2);
    return value;
}`)
	profile := cbackend.Stage0Profile()
	if runtime.GOOS == "windows" {
		profile = cbackend.Stage0WindowsMSVCProfile()
	}
	unit, err := cbackend.Emit(program, "main::main", profile)
	if err != nil {
		t.Fatal(err)
	}
	small, err := cbackend.EmitBuild(program, "main::main", profile, false)
	if err != nil || small.Unit.Source != unit.Source || len(small.Sources) != 0 {
		t.Fatalf("small build changed: %v", err)
	}
	parts, err := cbackend.EmitBuildContext(context.Background(), program, "main::main", profile, false, 256)
	if err != nil {
		t.Fatal(err)
	}
	if len(parts.Sources) < 3 || parts.Unit.Source != "" || parts.Header == "" {
		t.Fatal("missing function-boundary partitions")
	}
	// Header-only declarations omit local parameter names; definitions retain
	// them. This keeps exact C types and avoids redundant names in every unit.
	for _, line := range strings.Split(parts.Header, "\n") {
		if strings.Contains(line, " fy_fn_") && strings.HasSuffix(line, ");") && strings.Contains(line, " fy_l_") {
			t.Fatalf("shared prototype retained a definition-local name: %s", line)
		}
	}
	if !strings.Contains(strings.Join(parts.Sources, "\n"), " fy_l_") {
		t.Fatal("function definition lost its parameter names")
	}
	metadata := unit
	metadata.Source = ""
	if !reflect.DeepEqual(parts.Unit, metadata) {
		t.Fatal("split changes semantic metadata")
	}
	repeated, err := cbackend.EmitBuildContext(context.Background(), program, "main::main", profile, false, 256)
	if err != nil || !reflect.DeepEqual(parts, repeated) {
		t.Fatal("split output is nondeterministic", err)
	}
	joined := strings.Join(parts.Sources, "\n")
	if strings.Count(joined, "const uint8_t fy_static_bytes_") != 1 {
		t.Fatal("literal storage duplicated across translation units")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := cbackend.EmitBuildContext(ctx, program, "main::main", profile, false, 256); err != context.Canceled {
		t.Fatalf("cancellation lost: %v", err)
	}
	indexOf := func(name string) int {
		for i, nameFound := range parts.Unit.Functions {
			if nameFound == name {
				return i
			}
		}
		t.Fatalf("missing function %s", name)
		return -1
	}
	probe := fmt.Sprintf("int32_t same_literal(void) { return fy_fn_%d().data == fy_fn_%d().data; }", indexOf("main::first"), indexOf("main::second"))
	for _, optimization := range []string{"low", "optimized"} {
		t.Run(optimization, func(t *testing.T) { compileSplitForTest(t, parts, optimization, probe, 42) })
	}
}

func TestSmallBuildMatchesStreamingUnit(t *testing.T) {
	program := lowerSource(t, `module main;
unsafe extern C fn i32 visit(i32 value, unsafe extern C fn i32(i32,ptr mut void) callback, ptr mut void data);
@inline pub fn i32 combine(i32 a, i32 b) { return (a * 3 + b) / 2; }
unsafe fn i32 handle(i32 value, ptr mut void data) { return value + 1; }
resource Leaf { i32 value; cleanup { assert(this.value == 7); } }
resource Box { Leaf first; Leaf last; cleanup {} }
resource Outer { Box first; Box last; cleanup {} }
fn Box make() { return Box { .first = Leaf { .value = 7 }, .last = Leaf { .value = 7 } }; }
fn ref u8[] literal() { return b"shared"; }
fn i32 main() {
    let Outer tree = Outer { .first = make(), .last = make() };
    assert(literal()[0] == 115);
    assert(unsafe { visit(21, handle, null) } == 22);
    return combine(10, 12);
}`)
	profiles := []struct {
		name    string
		profile cbackend.Profile
	}{
		{name: "linux", profile: cbackend.Stage0Profile()},
		{name: "windows-msvc", profile: cbackend.Stage0WindowsMSVCProfile()},
		{name: "linux-drop-trace", profile: cbackend.Stage0DropTraceProfile()},
		{name: "windows-msvc-drop-trace", profile: cbackend.Stage0WindowsMSVCDropTraceProfile()},
	}
	for _, test := range profiles {
		t.Run(test.name, func(t *testing.T) {
			unit, err := cbackend.Emit(program, "main::main", test.profile)
			if err != nil {
				t.Fatal(err)
			}
			small, err := cbackend.EmitBuild(program, "main::main", test.profile, false)
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(small.Unit, unit) || small.Header != "" || len(small.Sources) != 0 {
				t.Fatal("small build changed the single-unit result")
			}
			ordered := []string{
				"const uint8_t fy_static_bytes_",
				"extern int32_t visit(",
				"static inline int32_t fy_fn_",
				"static void fy_drop_",
				"static int32_t fy_cb_",
				"int main(void)",
			}
			last := -1
			for _, required := range ordered {
				if !strings.Contains(unit.Source, required) {
					t.Fatalf("coverage case missing %q", required)
				}
				index := strings.Index(unit.Source[last+1:], required)
				if index < 0 {
					t.Fatalf("%q appears out of legacy emission order", required)
				}
				last += index + len(required)
			}
		})
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	for _, test := range profiles {
		if _, err := cbackend.EmitBuildContext(ctx, program, "main::main", test.profile, false, 2<<20); err != context.Canceled {
			t.Fatalf("%s cancellation lost: %v", test.name, err)
		}
	}
}

func TestPartitionedCallbackIdentity(t *testing.T) {
	program := lowerSource(t, `module main;
unsafe extern C fn i32 visit(i32 value, unsafe extern C fn i32(i32,ptr mut void) callback, ptr mut void data);
unsafe fn i32 handle(i32 value, ptr mut void data) { return value + 1; }
fn i32 first() { return unsafe { visit(21, handle, null) }; }
fn i32 second() { return unsafe { visit(19, handle, null) }; }
fn i32 main() { return first() + second(); }`)
	profile := cbackend.Stage0Profile()
	if runtime.GOOS == "windows" {
		profile = cbackend.Stage0WindowsMSVCProfile()
	}
	parts, err := cbackend.EmitBuildContext(context.Background(), program, "main::main", profile, false, 256)
	if err != nil {
		t.Fatal(err)
	}
	probe := `static fy_c_callback_i32_userdata previous;
int32_t visit(int32_t value, fy_c_callback_i32_userdata callback, void *data) {
    if (previous != 0 && previous != callback) { return -100; }
    previous = callback;
    return callback(value, data);
}`
	for _, optimization := range []string{"low", "optimized"} {
		t.Run(optimization, func(t *testing.T) { compileSplitForTest(t, parts, optimization, probe, 42) })
	}
}

func TestPartitionedDropHelperPrototypes(t *testing.T) {
	program := lowerSource(t, `module main;
unsafe extern C fn void observe(i32 value);
unsafe extern C fn i32 finish();
resource Leaf{i32 value;cleanup{unsafe{observe(this.value);};}}
resource Box{Leaf inner;cleanup{unsafe{observe(9);};}}
fn i32 main(){{let box=Box{.inner=Leaf{.value=4}};}return unsafe{finish()};}`)
	profile := cbackend.Stage0Profile()
	if runtime.GOOS == "windows" {
		profile = cbackend.Stage0WindowsMSVCProfile()
	}
	unit, err := cbackend.Emit(program, "main::main", profile)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(unit.Source, " *fy_value);") || !strings.Contains(unit.Source, " fy_a_0") {
		t.Fatal("single-unit legacy drop prototype changed or fixture has no nested cleanup")
	}
	parts, err := cbackend.EmitBuildContext(t.Context(), program, "main::main", profile, false, 256)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(parts.Header, "void fy_drop_") || strings.Contains(parts.Header, "fy_value") || strings.Contains(parts.Header, " fy_a_") || !strings.Contains(strings.Join(parts.Sources, "\n"), "*fy_value) {") {
		t.Fatal("shared declaration must omit names while cleanup definitions retain them")
	}
	probe := `static int32_t trace;
void observe(int32_t value){trace=trace*10+value;}
int32_t finish(void){return trace==94?42:1;}`
	for _, optimization := range []string{"low", "optimized"} {
		t.Run(optimization, func(t *testing.T) { compileSplitForTest(t, parts, optimization, probe, 42) })
	}
}

func TestPartitionIncludesInByteBudget(t *testing.T) {
	program := lowerSource(t, `module main;
fn i32 first() { return 10; }
fn i32 second() { return 12; }
fn i32 main() { return first() + second(); }`)
	for _, profile := range []cbackend.Profile{cbackend.Stage0Profile(), cbackend.Stage0WindowsMSVCProfile()} {
		// A one-byte target keeps each indivisible function separate. Derive
		// the boundary from real emitted bodies rather than frozen C spelling.
		separate, err := cbackend.EmitBuildContext(t.Context(), program, "main::main", profile, false, 1)
		if err != nil {
			t.Fatal(err)
		}
		const include = "#include \"fy_bootstrap.h\"\n"
		if len(separate.Sources) < 3 {
			t.Fatal("missing independent function bodies")
		}
		first := strings.TrimPrefix(separate.Sources[0], include)
		second := strings.TrimPrefix(separate.Sources[1], include)
		boundary := len(include) + len(first) + len(second)
		for _, delta := range []int{-1, 0, 1} {
			budget := boundary + delta
			parts, err := cbackend.EmitBuildContext(t.Context(), program, "main::main", profile, false, budget)
			if err != nil {
				t.Fatal(err)
			}
			paired := false
			for _, source := range parts.Sources {
				paired = paired || strings.Contains(source, first) && strings.Contains(source, second)
				if len(source) <= budget {
					continue
				}
				// A single large function still cannot be split. Only packing
				// multiple bodies must respect the target including its prefix.
				indivisible := false
				for _, original := range separate.Sources {
					indivisible = indivisible || source == original
				}
				if !indivisible {
					t.Fatalf("budget %d: grouped source has %d bytes", budget, len(source))
				}
			}
			if paired != (delta >= 0) {
				t.Fatalf("budget %d: adjacent bodies paired = %t", budget, paired)
			}
		}
	}
}

// Compile actual separate files with the host's strict consumer; concatenation
// would miss duplicate linkage, cross-unit literal identity and header warnings.
func compileSplitForTest(t *testing.T, parts cbackend.BuildUnit, profile, extra string, want int) {
	t.Helper()
	root := t.TempDir()
	write := func(name, text string) {
		if err := os.WriteFile(filepath.Join(root, name), []byte(text), 0600); err != nil {
			t.Fatal(err)
		}
	}
	write("fy_bootstrap.h", parts.Header)
	var sources []string
	for i, source := range parts.Sources {
		name := fmt.Sprintf("part_%d.c", i)
		write(name, source)
		sources = append(sources, name)
	}
	support, err := cruntime.ScalarSource(cdriver.NativeTrapExitBase)
	if err != nil {
		t.Fatal(err)
	}
	write("runtime.c", support)
	sources = append(sources, "runtime.c")
	if extra != "" {
		write("probe.c", "#include \"fy_bootstrap.h\"\n"+extra+"\n")
		sources = append(sources, "probe.c")
	}
	var compiler string
	var args, env []string
	if runtime.GOOS == "windows" {
		cc, err := cdriver.ResolveWindowsHostMSVC()
		if err != nil {
			t.Fatal(err)
		}
		compiler, env = cc.Path, cc.Environment
		opt := "/Od"
		if profile == "optimized" {
			opt = "/O2"
		}
		args = []string{"/nologo", "/std:c11", "/TC", "/W4", "/WX", "/MT", opt}
		args = append(args, sources...)
		args = append(args, "/Fe:program.exe", "/link", "/INCREMENTAL:NO")
	} else if runtime.GOOS == "linux" {
		cc, err := cdriver.Resolve(target.ID(parts.Unit.Target), "gcc")
		if err != nil {
			t.Fatal(err)
		}
		compiler = cc.Path
		opt := "-O0"
		if profile == "optimized" {
			opt = "-O2"
		}
		args = []string{"-std=c11", "-pedantic-errors", "-Wall", "-Wextra", "-Werror", "-fno-common", opt}
		args = append(args, sources...)
		args = append(args, "-o", "program")
	} else {
		t.Skip("requires a native C host")
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	cmd := exec.CommandContext(ctx, compiler, args...)
	cmd.Dir = root
	if env != nil {
		cmd.Env = env
	}
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("compile partitions: %v\n%s", err, output)
	}
	name := "program"
	if runtime.GOOS == "windows" {
		name += ".exe"
	}
	output, err := exec.CommandContext(ctx, filepath.Join(root, name)).CombinedOutput()
	if exit, ok := err.(*exec.ExitError); !ok || exit.ExitCode() != want || len(output) != 0 {
		t.Fatalf("split program: %v %q", err, output)
	}
}

func TestPartitionedOrdinaryCallback(t *testing.T) {
	program := lowerSource(t, `module main;
unsafe extern C fn i32 ordinary_visit(unsafe extern C fn i32(i32,i32,ptr mut void) cb,ptr mut void data);
unsafe fn i32 ordinary_handle(i32 a,i32 b,ptr mut void data){return a+b;}
fn i32 first(){return unsafe{ordinary_visit(ordinary_handle,null)};}
fn i32 second(){return unsafe{ordinary_visit(ordinary_handle,null)};}
fn i32 main(){return first()+second();}`)
	profile := cbackend.Stage0Profile()
	if runtime.GOOS == "windows" {
		profile = cbackend.Stage0WindowsMSVCProfile()
	}
	parts, err := cbackend.EmitBuildContext(context.Background(), program, "main::main", profile, false, 128)
	if err != nil {
		t.Fatal(err)
	}
	probe := `static int32_t (*previous)(int32_t,int32_t,void *);
int32_t ordinary_visit(int32_t (*cb)(int32_t,int32_t,void *),void *data){if(previous&&previous!=cb){return -100;}previous=cb;return cb(10,11,data);}`
	for _, opt := range []string{"low", "optimized"} {
		t.Run(opt, func(t *testing.T) { compileSplitForTest(t, parts, opt, probe, 42) })
	}
}

func TestPartitionedInlineBody(t *testing.T) {
	program := lowerSource(t, `module main;
@inline pub fn i32 combine(i32 a,i32 b){return (a*3+b)/2;}
fn i32 first(){return combine(10,12);}
fn i32 second(){return combine(10,12);}
fn i32 main(){return first()+second();}`)
	profile := cbackend.Stage0Profile()
	if runtime.GOOS == "windows" {
		profile = cbackend.Stage0WindowsMSVCProfile()
	}
	parts, err := cbackend.EmitBuildContext(context.Background(), program, "main::main", profile, false, 128)
	if err != nil {
		t.Fatal(err)
	}
	if len(parts.Sources) < 2 || !strings.Contains(parts.Header, "static inline int32_t") || strings.Contains(strings.Join(parts.Sources, ""), "static inline int32_t") {
		t.Fatal("inline definition must be provided once in the shared header")
	}
	for _, opt := range []string{"low", "optimized"} {
		t.Run(opt, func(t *testing.T) { compileSplitForTest(t, parts, opt, "", 42) })
	}
}
