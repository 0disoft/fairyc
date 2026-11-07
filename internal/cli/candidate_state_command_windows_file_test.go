//go:build windows

package cli

import (
	"bufio"
	"bytes"
	"context"
	"crypto/sha256"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"golang.org/x/sys/windows"
)

const stateCommandWindowsFileProbe = `module main;
import module_state_command_windows_file as files;import windows_cancel as cancel;import std::process as process;import std::bytes as bytes;import std::io as io;
@link_name("GetCurrentProcess") unsafe extern C fn usize current();
@link_name("GetProcessHandleCount") unsafe extern C fn i32 handles(usize process,ptr mut u32 count);
fn u32 handle_count(){var u32 count=0;assert(unsafe{handles(current(),ref mut count as ptr mut u32)}!=0);return count;}
fn usize argument_number(usize at){var u8[32] raw=[0;32];let size=switch(process::arg(at,ref mut raw)){case .ok(p):{p}case .err(_):{assert(false);0}};var usize n=0;var usize i=0;while(i<size){assert(raw[i]>=48 && raw[i]<=57);n=n*10+(raw[i]-48) as usize;i+=1;}return n;}
@effects(console)fn void number(usize n){assert(io::write_usize(n).is_ok && io::write("\n").is_ok);}
@effects(console)fn void blob(ref u8[] raw){number(raw.len);assert(io::write_bytes(raw).is_ok && io::write("\n").is_ok);}
@effects(filesystem,console,environment)fn i32 main(){
 let maximum=argument_number(1);let initial=argument_number(2);var u8[512] event_name=[0;512];let size=switch(process::arg(3,ref mut event_name)){case .ok(p):{p}case .err(_):{return 1;}};let ref u8[] name=ref event_name;
 let token=switch(cancel::open(name[..size])){case .ok(p):{p}case .err(_):{return 2;}};
 var path=bytes::empty();var u8[4096] chunk=[0;4096];while(true){let n=switch(io::read_stdin(ref mut chunk)){case .ok(p):{p}case .err(_):{return 3;}};if(n==0){break;}let ref u8[] raw=ref chunk;assert(bytes::append(ref mut path,raw[..n]).is_ok && bytes::len(ref path)<=131072);}
 let original=switch(bytes::copy(bytes::view(ref path))){case .ok(p):{p}case .err(_):{return 4;}};
 let before=handle_count();var usize i=0;while(i<4){var work=initial;
  switch(files::load(bytes::view(ref path),maximum,ref token,ref mut work)){
   case .ok(data):{assert(handle_count()==before);number(0);number(0);number(0);number(work);number(before as usize);number(handle_count() as usize);blob(bytes::view(ref data));}
   case .err(e):{assert(handle_count()==before);number(e.stage as usize);number(e.code as usize);number(e.os_code as usize);number(work);number(before as usize);number(handle_count() as usize);blob(b"");}
  };assert(bytes::equal(bytes::view(ref path),bytes::view(ref original)));assert(handle_count()==before);i+=1;
 }return 0;
}`

func TestCandidateStateCommandWindowsFile(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("..", "..", "examples", "lexer_candidate", "src", "module_state_command_windows_file.fy"))
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("module_state_command_windows_file.fy sha256=%x; probe sha256=%x", sha256.Sum256(data), sha256.Sum256([]byte(stateCommandWindowsFileProbe)))
	directory := filepath.Join(t.TempDir(), "한글 폴더 with spaces")
	if err = os.Mkdir(directory, 0700); err != nil {
		t.Fatal(err)
	}
	payload := bytes.Repeat([]byte{0, 255, 13, 10, 42, 99, 0}, 9363)[:65537]
	for name, content := range map[string][]byte{"empty.bin": {}, "exact.bin": payload[:32], "여러 청크.bin": payload} {
		if err = os.WriteFile(filepath.Join(directory, name), content, 0600); err != nil {
			t.Fatal(err)
		}
	}
	type trial struct {
		name, path          string
		maximum, initial    uint64
		stage, code, native uint64
		canceled            bool
		want                []byte
	}
	trials := []trial{
		{name: "empty", path: filepath.Join(directory, "empty.bin"), maximum: 1, initial: 11},
		{name: "exact", path: filepath.Join(directory, "exact.bin"), maximum: 32, initial: 11, want: payload[:32]},
		{name: "above", path: filepath.Join(directory, "exact.bin"), maximum: 31, initial: 11, stage: 6, code: 1002},
		{name: "unicode-multiple-chunks", path: filepath.Join(directory, "여러 청크.bin"), maximum: 65537, initial: 11, want: payload},
		{name: "relative-bare", path: "exact.bin", maximum: 32, initial: 11, want: payload[:32]},
		{name: "relative-parent", path: ".\\exact.bin", maximum: 32, initial: 11, want: payload[:32]},
		{name: "forward-separators", path: filepath.ToSlash(filepath.Join(directory, "exact.bin")), maximum: 32, initial: 11, want: payload[:32]},
		{name: "maximum-option", path: "exact.bin", maximum: 16777216, initial: 11, want: payload[:32]},
		{name: "directory-nonregular", path: directory, maximum: 32, initial: 11, stage: 4, native: 1},
		{name: "missing", path: "missing.bin", maximum: 32, initial: 11, stage: 4, native: 2},
		{name: "missing-parent", path: "missing-parent\\file.bin", maximum: 32, initial: 11, stage: 3, native: 2},
		{name: "pre-canceled-missing", path: "missing.bin", maximum: 32, initial: 11, stage: 2, code: 1009, canceled: true},
		{name: "zero-limit-precedence", path: "exact.bin", initial: 11, stage: 1, code: 1001, canceled: true},
		{name: "invalid-limit", path: "exact.bin", maximum: 16777217, initial: 11, stage: 1, code: 1001},
		{name: "work-bound-precedence", path: "exact.bin", maximum: 32, initial: 16777216, stage: 1, code: 1002, canceled: true},
		{name: "work-overdrawn", path: "exact.bin", maximum: 32, initial: 16777217, stage: 1, code: 1002},
		{name: "empty-path", maximum: 32, initial: 11, stage: 1, code: 1001},
		{name: "nul", path: "exact.bin\x00other", maximum: 32, initial: 11, stage: 1, code: 1001},
		{name: "invalid-utf8", path: "bad\xff.bin", maximum: 32, initial: 11, stage: 1, code: 1001},
		{name: "ads", path: "exact.bin:secret", maximum: 32, initial: 11, stage: 1, code: 1001},
		{name: "drive-relative", path: filepath.VolumeName(directory) + "exact.bin", maximum: 32, initial: 11, stage: 1, code: 1001},
		{name: "trailing-slash", path: directory + "\\", maximum: 32, initial: 11, stage: 1, code: 1001},
		{name: "overlong-leaf", path: strings.Repeat("가", 256), maximum: 32, initial: 11, stage: 1, code: 1001},
	}
	// A root-parent split that accidentally drops the slash opens C: (the
	// child's cwd) and wrongly finds this same basename. No drive-root writes.
	rootLeaf := fmt.Sprintf("fairyc-root-split-%x.bin", sha256.Sum256([]byte(directory)))
	if err = os.WriteFile(filepath.Join(directory, rootLeaf), payload[:32], 0600); err != nil {
		t.Fatal(err)
	}
	rootPath := filepath.VolumeName(directory) + "\\" + rootLeaf
	if _, err = os.Stat(rootPath); !os.IsNotExist(err) {
		t.Fatalf("drive-root fixture must be absent: %v", err)
	}
	trials = append(trials, trial{name: "drive-root-parent", path: rootPath, maximum: 32, initial: 11, stage: 4, native: 2})
	link := filepath.Join(directory, "reparse.bin")
	if err = os.Symlink(filepath.Join(directory, "exact.bin"), link); err == nil {
		trials = append(trials, trial{name: "reparse-leaf", path: link, maximum: 32, initial: 11, stage: 4, native: 1})
	} else {
		t.Logf("reparse-leaf unavailable: local symlink creation refused: %v", err)
	}
	candidateWindowsCancellationProbe(t, "module_state_command_windows_file", stateCommandWindowsFileProbe, func(t *testing.T, image string) {
		run := func(t *testing.T, test trial) uint64 {
			t.Helper()
			name := fmt.Sprintf("Local\\fairyc-cancel-%x", sha256.Sum256([]byte(t.Name()+directory)))
			native, err := windows.UTF16PtrFromString(name)
			if err != nil {
				t.Fatal(err)
			}
			signaled := uint32(0)
			if test.canceled {
				signaled = 1
			}
			event, err := windows.CreateEvent(nil, 1, signaled, native)
			if err != nil {
				t.Fatal(err)
			}
			defer windows.CloseHandle(event)
			ctx, cancel := context.WithTimeout(t.Context(), 10*time.Second)
			defer cancel()
			command := exec.CommandContext(ctx, image, strconv.FormatUint(test.maximum, 10), strconv.FormatUint(test.initial, 10), name)
			command.Dir = directory
			command.Stdin = strings.NewReader(test.path)
			var output, stderr bytes.Buffer
			command.Stdout, command.Stderr = &output, &stderr
			if err = command.Run(); err != nil || stderr.Len() != 0 {
				t.Fatalf("native loader err=%v stdout=%q stderr=%q", err, &output, &stderr)
			}
			r := graphResponseReader{t, bufio.NewReader(bytes.NewReader(output.Bytes()))}
			var charged uint64
			for i := 0; i < 4; i++ {
				stage, code, osCode, work, before, after := r.number(), r.number(), r.number(), r.number(), r.number(), r.number()
				got := r.blob()
				if stage != test.stage || code != test.code || before == 0 || before != after || work < test.initial || work > 16777216 && test.initial <= 16777216 || !bytes.Equal(got, test.want) {
					t.Fatalf("stage/code/os/work/handles=%d/%d/%d/%d/%d:%d bytes=%d want stage/code=%d/%d bytes=%d", stage, code, osCode, work, before, after, len(got), test.stage, test.code, len(test.want))
				}
				if test.native == 1 && osCode == 0 || test.native > 1 && osCode != test.native || test.native == 0 && osCode != 0 {
					t.Fatalf("native OS cause=%d want=%d (1 means any nonzero original cause)", osCode, test.native)
				}
				if (test.stage == 1 || test.canceled) && work != test.initial {
					t.Fatal("preflight/pre-cancel charged caller work", work, test.initial)
				}
				if i != 0 && work != charged {
					t.Fatal("non-deterministic cumulative scalar charge", work, charged)
				}
				charged = work
			}
			remaining, err := io.ReadAll(r.r)
			if err != nil || len(remaining) != 0 {
				t.Fatal("unexpected native output suffix", remaining, err)
			}
			t.Logf("4 retained-handle loads: stage=%d code=%d work=%d input unchanged; handle count stable", test.stage, test.code, charged)
			return charged
		}
		for _, test := range trials {
			t.Run(test.name, func(t *testing.T) { run(t, test) })
		}
		t.Run("exact-and-one-short-work", func(t *testing.T) {
			test := trial{name: "budget", path: "exact.bin", maximum: 32, initial: 11, want: payload[:32]}
			cost := run(t, test) - test.initial
			test.initial = 16777216 - cost
			if got := run(t, test); got != 16777216 {
				t.Fatal("exact caller work did not reach limit", got)
			}
			test.initial++
			test.stage, test.code, test.want = 6, 1002, nil
			run(t, test)
		})
	})
}
