package cbackend_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/cbackend"
)

func TestPrivateVectorRuntimeSymbolBoundary(t *testing.T) {
	for _, entry := range []struct{ name, result, parameters, arguments string }{
		{"initialize", "u32", "ptr mut void storage, usize size, usize alignment", "null, 0, 0"},
		{"release_empty", "u32", "ptr mut void storage", "null"},
		{"reserve", "u32", "ptr mut void storage, usize capacity", "null, 0"},
		{"prepare_push", "u32", "ptr mut void storage, usize size, ptr mut void output", "null, 0, null"},
		{"commit_push", "u32", "ptr mut void storage, ptr mut void slot, usize size", "null, null, 0"},
		{"pop_slot", "ptr mut void", "ptr mut void storage, usize size", "null, 0"},
	} {
		t.Run(entry.name, func(t *testing.T) {
			text := fmt.Sprintf(`module main;
@link_name("fy_private_vector_%s_v0") unsafe extern C fn %s operation(%s);
fn i32 main() { let %s result = unsafe { operation(%s) }; return 42; }`, entry.name, entry.result, entry.parameters, entry.result, entry.arguments)
			program := lowerSource(t, text)
			if _, err := cbackend.Emit(program, "main::main", cbackend.Stage0WindowsMSVCProfile()); err == nil {
				t.Fatal("ordinary backend admitted private vector symbol")
			}
			if _, err := cbackend.EmitStorageTransfers(program, "main::main", cbackend.Stage0WindowsMSVCProfile()); err != nil {
				t.Fatal(err)
			}
			if _, err := cbackend.EmitStorageTransfers(program, "main::main", cbackend.Stage0Profile()); err == nil {
				t.Fatal("unvalidated target admitted vector runtime")
			}
			wrong := strings.Replace(text, "fn "+entry.result+" operation", "fn i32 operation", 1)
			wrong = strings.Replace(wrong, "let "+entry.result+" result", "let i32 result", 1)
			if _, err := cbackend.EmitStorageTransfers(lowerSource(t, wrong), "main::main", cbackend.Stage0WindowsMSVCProfile()); err == nil {
				t.Fatal("incorrect result ABI admitted")
			}
			wrong = strings.ReplaceAll(text, "ptr mut void storage", "ptr void storage")
			if _, err := cbackend.EmitStorageTransfers(lowerSource(t, wrong), "main::main", cbackend.Stage0WindowsMSVCProfile()); err == nil {
				t.Fatal("incorrect parameter ABI admitted")
			}
		})
	}
	program := lowerSource(t, `module main;
@link_name("fy_private_vector_unknown_v0") unsafe extern C fn u32 operation(ptr mut void storage);
fn i32 main() { let u32 status = unsafe { operation(null) }; return 42; }`)
	if _, err := cbackend.EmitStorageTransfers(program, "main::main", cbackend.Stage0WindowsMSVCProfile()); err == nil {
		t.Fatal("unknown vector symbol admitted")
	}
}
