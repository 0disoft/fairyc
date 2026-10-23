package cli

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"slices"
	"testing"
	"time"

	"github.com/0disoft/fairyc/internal/cbackend"
	"github.com/0disoft/fairyc/internal/cdriver"
	"github.com/0disoft/fairyc/internal/nativesupport"
	"github.com/0disoft/fairyc/internal/target"
	"github.com/0disoft/fairyc/internal/workspacecompiler"
)

const sdkFileBufferSource = `module main;
import std::bytes as bytes;
import std::fs as fs;

fn usize count(Result<usize, fs::ReadAppendError> result) {
    return switch (result) {
        case .ok(n): { n }
        case .err(_): { assert(false); 0 }
    };
}

@effects(filesystem)
fn void read_small(fs::File input) {
    var fs::File file = input;
    var bytes::Buffer buffer = bytes::empty();
    assert(bytes::append(ref mut buffer, b"pre").is_ok);
    let usize capacity = bytes::capacity(ref buffer);
    assert(count(fs::read_append(ref mut file, ref mut buffer, 0)) == 0);
    assert(bytes::capacity(ref buffer) == capacity);
    assert(bytes::equal(bytes::view(ref buffer), b"pre"));
    assert(switch (fs::position(ref mut file)) { case .ok(n): { n == 0 } case .err(_): { false } });
    assert(count(fs::read_append(ref mut file, ref mut buffer, 2)) == 2);
    assert(bytes::equal(bytes::view(ref buffer), b"preA\x00"));
    assert(count(fs::read_append(ref mut file, ref mut buffer, 8)) == 2);
    assert(bytes::equal(bytes::view(ref buffer), b"preA\x00\xff\n"));
    assert(count(fs::read_to_end(ref mut file, ref mut buffer)) == 0);
    assert(bytes::equal(bytes::view(ref buffer), b"preA\x00\xff\n"));
    assert(fs::close(file).is_ok);
}

@effects(filesystem)
fn void read_large(fs::File input) {
    var fs::File file = input;
    var bytes::Buffer buffer = bytes::empty();
    assert(bytes::append(ref mut buffer, b"P").is_ok);
    assert(count(fs::read_to_end(ref mut file, ref mut buffer)) == 65539);
    let ref u8[] contents = bytes::view(ref buffer);
    assert(contents.len == 65540 && contents[0] == 80);
    var usize at = 1;
    while (at <= 65536) { assert(contents[at] == 90); at += 1; }
    assert(contents[65537] == 0 && contents[65538] == 255 && contents[65539] == 10);
    assert(fs::close(file).is_ok);
}

@effects(filesystem)
fn void write_only(fs::File output) {
    var fs::File file = output;
    assert(switch (fs::write_all(ref mut file, b"A\x00\xff\n")) { case .ok(n): { n == 4 } case .err(_): { false } });
    var bytes::Buffer buffer = bytes::empty();
    assert(bytes::append(ref mut buffer, b"saved").is_ok);
    assert(switch (fs::read_append(ref mut file, ref mut buffer, 32)) {
        case .ok(_): { false }
        case .err(error): {
            fs::read_append_error_file_status(error) == 1 &&
            fs::read_append_error_os_code(error) != 0 &&
            fs::read_append_error_allocation_status(error) == 0
        }
    });
    assert(bytes::equal(bytes::view(ref buffer), b"saved"));
    assert(switch (fs::position(ref mut file)) { case .ok(n): { n == 4 } case .err(_): { false } });
    assert(fs::close(file).is_ok);
}

@effects(filesystem)
fn void make_large(fs::File output) {
    var fs::File file = output;
    let u8[1024] block = [90; 1024];
    var usize at = 0;
    while (at < 64) {
        assert(switch (fs::write_all(ref mut file, ref block)) { case .ok(n): { n == 1024 } case .err(_): { false } });
        at += 1;
    }
    assert(switch (fs::write_all(ref mut file, b"\x00\xff\n")) { case .ok(n): { n == 3 } case .err(_): { false } });
    assert(fs::close(file).is_ok);
}

@effects(filesystem)
fn i32 main() {
    switch (fs::create_new(c"small.dat")) { case .ok(file): { write_only(file); } case .err(_): { return 1; } };
    switch (fs::open_read(c"small.dat")) { case .ok(file): { read_small(file); } case .err(_): { return 2; } };
    switch (fs::create_new(c"large.dat")) { case .ok(file): { make_large(file); } case .err(_): { return 3; } };
    switch (fs::open_read(c"large.dat")) { case .ok(file): { read_large(file); } case .err(_): { return 4; } };
    assert(fs::remove_file(c"small.dat").is_ok);
    assert(fs::remove_file(c"large.dat").is_ok);
    return 0;
}
`

func sdkFileBufferUnit(t *testing.T, profile cbackend.Profile) cbackend.Unit {
	t.Helper()
	checked, err := workspacecompiler.Check(checkedIntegerConversionWorkspace(t, sdkFileBufferSource))
	if err != nil || !checked.Verified {
		t.Fatalf("SDK File Buffer check: %v %+v %+v", err, checked.Semantic.Diagnostics, checked.Semantic.Unsupported)
	}
	unit, err := cbackend.EmitStorageTransfers(checked.MIR, tokenScannerEntry(t, checked), profile)
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Contains(unit.ForeignSymbols, "fy_std_file_read_append_v0") || slices.Contains(unit.ForeignSymbols, "fy_std_windows_file_read_append_v0") {
		t.Fatalf("SDK did not select portable File Buffer: %v", unit.ForeignSymbols)
	}
	return unit
}

func TestSDKFileBufferEmission(t *testing.T) {
	for _, profile := range []cbackend.Profile{cbackend.Stage0Profile(), cbackend.Stage0WindowsMSVCProfile()} {
		t.Run(string(profile.Target), func(t *testing.T) {
			unit := sdkFileBufferUnit(t, profile)
			if _, err := nativesupport.Source(profile.Target, unit.ForeignSymbols, nativesupport.Options{}); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestSDKFileBufferExecution(t *testing.T) {
	if runtime.GOARCH != "amd64" || (runtime.GOOS != "linux" && runtime.GOOS != "windows") {
		t.Skip("requires Linux GNU or Windows MSVC amd64")
	}
	if runtime.GOOS == "linux" {
		runLinuxSDKHelperExecution(t, sdkFileBufferUnit(t, cbackend.Stage0Profile()))
		return
	}
	unit := sdkFileBufferUnit(t, cbackend.Stage0WindowsMSVCProfile())
	support, err := nativesupport.Source(target.X8664WindowsMSVC, unit.ForeignSymbols, nativesupport.Options{})
	if err != nil {
		t.Fatal(err)
	}
	tc, err := cdriver.ResolveWindowsHostMSVC()
	if err != nil {
		t.Fatal(err)
	}
	for _, profile := range []cdriver.Profile{cdriver.LowProfile, cdriver.OptimizedProfile} {
		t.Run("msvc/"+profile.Name, func(t *testing.T) {
			artifact, err := cdriver.BuildWindowsMSVCTarget(tc, profile, unit.Source, support)
			if err != nil {
				t.Fatal(err)
			}
			root := t.TempDir()
			image := filepath.Join(root, "file-buffer.exe")
			if err := os.WriteFile(image, artifact.Bytes, 0700); err != nil {
				t.Fatal(err)
			}
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			command := exec.CommandContext(ctx, image)
			command.Dir = root
			if output, err := command.CombinedOutput(); err != nil || len(output) != 0 {
				t.Fatalf("SDK File Buffer: %v output=%q", err, output)
			}
			for _, name := range []string{"small.dat", "large.dat"} {
				if _, err := os.Stat(filepath.Join(root, name)); !os.IsNotExist(err) {
					t.Fatalf("file was not removed: %s: %v", name, err)
				}
			}
		})
	}
}
