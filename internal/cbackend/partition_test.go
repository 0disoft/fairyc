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
		write("probe.c", "#include \"fy_bootstrap.h\"\n"+extra)
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
