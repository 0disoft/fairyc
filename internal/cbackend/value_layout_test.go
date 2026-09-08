package cbackend_test

import (
	"testing"

	"github.com/0disoft/fairyc/internal/cbackend"
	"github.com/0disoft/fairyc/internal/hir"
)

func TestStorageLayoutMatchesCRepresentation(t *testing.T) {
	t.Parallel()
	program := lowerSource(t, `module main;
struct Padded { u8 tag; u32 value; }
resource Owner { Padded payload; ptr mut void handle; cleanup {} }
enum Choice { Empty; Value { u32 value; }; }
fn i32 main() { return 0; }
`)
	for _, profile := range []cbackend.Profile{cbackend.Stage0Profile(), cbackend.Stage0WindowsMSVCProfile()} {
		for _, item := range []struct {
			typ, representation hir.Type
			size, alignment     uint64
		}{
			{hir.TypeBool, hir.TypeI32, 4, 4},
			{hir.TypeU8, hir.TypeU8, 1, 1},
			{hir.TypeI32, hir.TypeI32, 4, 4},
			{hir.TypeU32, hir.TypeU32, 4, 4},
			{hir.TypeUSize, hir.TypeUSize, 8, 8},
			{"main::Padded", "main::Padded", 8, 4},
			{"main::Owner", "main::Owner", 16, 8},
			{"main::Choice", "main::Choice", 8, 4},
		} {
			got, err := cbackend.StorageLayout(program, item.typ, profile)
			if err != nil || got.Type != item.typ || got.Representation != item.representation || got.Target != profile.Target || got.SizeBytes != item.size || got.AlignmentBytes != item.alignment {
				t.Fatalf("%s on %s: %+v %v", item.typ, profile.Target, got, err)
			}
		}
	}
}

func TestStorageLayoutRejectsUnsupportedOrUnverifiedInputs(t *testing.T) {
	t.Parallel()
	program := lowerSource(t, "module main; fn i32 main() { return 0; }")
	profile := cbackend.Stage0WindowsMSVCProfile()
	for _, typ := range []hir.Type{hir.TypeVoid, "i64", hir.TypeStr, "main::Missing", hir.SharedBorrow(hir.TypeI32), hir.Option(hir.TypeI32), hir.RawPointer(hir.TypeVoid, false)} {
		if got, err := cbackend.StorageLayout(program, typ, profile); err == nil || got != (cbackend.ValueStorageLayout{}) {
			t.Fatalf("unsupported %s: %+v %v", typ, got, err)
		}
	}
	badProfile := profile
	badProfile.Dialect = "unadmitted"
	if _, err := cbackend.StorageLayout(program, hir.TypeI32, badProfile); err == nil {
		t.Fatal("unsupported profile admitted")
	}
	program.Functions[0].Blocks = nil
	if got, err := cbackend.StorageLayout(program, hir.TypeI32, profile); err == nil || got != (cbackend.ValueStorageLayout{}) {
		t.Fatalf("unverified MIR: %+v %v", got, err)
	}
}
