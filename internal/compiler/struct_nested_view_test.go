package compiler_test

import (
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/cbackend"
	"github.com/0disoft/fairyc/internal/compiler"
	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/mir"
	"github.com/0disoft/fairyc/internal/mircodec"
)

func nestedStorageViewSource(source string) string {
	return strings.Replace(source, "struct Token {u32 kind; usize offset; i64 value; bool active;}", "struct Span {usize start; usize end;} struct Token {u32 kind; Span position; i64 value; bool active;}", 1)
}

func TestStructNestedStorageMIR(t *testing.T) {
	source := nestedStorageViewSource(structStorageViewSource)
	analysis, err := analyzeStorageBorrow(t, source)
	if err != nil {
		t.Fatal(err)
	}
	checked, err := compiler.LowerAnalyzedStorageTransfers(analysis.Analysis)
	if err != nil || !checked.Verified {
		t.Fatalf("lower: %v", err)
	}
	encoded, err := mircodec.EncodeStorageTransfers(checked.MIR, 1<<20)
	if err != nil {
		t.Fatal(err)
	}
	for _, mutation := range []string{"empty", "missing", "borrow", "pointer", "resource", "enum", "zero-array", "oversize-array", "nested-array", "option", "result", "self-cycle", "indirect-cycle"} {
		t.Run(mutation, func(t *testing.T) {
			program, err := mircodec.DecodeStorageTransfers(encoded, 1<<20)
			if err != nil {
				t.Fatal(err)
			}
			var token hir.Type
			for _, s := range program.StructTypes {
				if strings.HasSuffix(string(s.Type), "::Token") {
					token = s.Type
				}
			}
			found := false
			for i := range program.StructTypes {
				s := &program.StructTypes[i]
				if !strings.HasSuffix(string(s.Type), "::Span") {
					continue
				}
				found = true
				switch mutation {
				case "empty":
					s.Fields = nil
				case "missing":
					s.Fields[0].Type = hir.Type("main::Missing")
				case "borrow":
					s.Fields[0].Type = hir.SharedBorrow(hir.TypeI32)
				case "pointer":
					s.Fields[0].Type = hir.RawPointer(hir.TypeI32, true)
				case "resource":
					s.Fields[0].Type = program.ResourceTypes[0].Type
				case "enum":
					s.Fields[0].Type = hir.Type("main::Tag")
					program.EnumTypes = append(program.EnumTypes, mir.EnumType{Type: hir.Type("main::Tag"), Name: "Tag"})
				case "zero-array":
					s.Fields[0].Type = hir.FixedArray(hir.TypeI32, 0)
				case "oversize-array":
					s.Fields[0].Type = hir.FixedArray(hir.TypeI32, 4097)
				case "nested-array":
					s.Fields[0].Type = hir.FixedArray(hir.FixedArray(hir.TypeI32, 2), 2)
				case "option":
					s.Fields[0].Type = hir.Option(hir.TypeI32)
				case "result":
					s.Fields[0].Type = hir.Result(hir.TypeI32, hir.TypeU32)
				case "self-cycle":
					s.Fields[0].Type = s.Type
				case "indirect-cycle":
					s.Fields[0].Type = token
				}
			}
			if !found {
				t.Fatal("codec lost nested definition")
			}
			failures := mir.VerifyStorageTransfers(program)
			viewFailure := false
			for _, failure := range failures {
				if strings.Contains(failure.Detail, "struct view requires") {
					viewFailure = true
				}
			}
			if !viewFailure {
				t.Fatalf("nested leaf bypassed view verifier: %+v", failures)
			}
			if _, err := mircodec.EncodeStorageTransfers(program, 1<<20); err == nil {
				t.Fatal("invalid nested graph crossed codec")
			}
		})
	}
}

// ADR 0468 admits bounded inline scalar arrays even through nested plain structs.
// Keep positive source-to-codec/backend coverage beside the invalid graph mutations.
func TestStructNestedStorageArrayFieldMIR(t *testing.T) {
	for _, test := range []struct{ name, field string }{
		{"minimum", "bool[1]"},
		{"previous-rejection-shape", "i32[2]"},
		{"maximum", "i64[4096]"},
	} {
		t.Run(test.name, func(t *testing.T) {
			source := strings.Replace(nestedStorageViewSource(structStorageViewSource), "usize start", test.field+" start", 1)
			analysis, err := analyzeStorageBorrow(t, source)
			if err != nil {
				t.Fatal(err)
			}
			checked, err := compiler.LowerAnalyzedStorageTransfers(analysis.Analysis)
			if err != nil || !checked.Verified {
				t.Fatalf("lower admitted nested array field: %v", err)
			}
			encoded, err := mircodec.EncodeStorageTransfers(checked.MIR, 1<<20)
			if err != nil {
				t.Fatal(err)
			}
			program, err := mircodec.DecodeStorageTransfers(encoded, 1<<20)
			if err != nil {
				t.Fatal(err)
			}
			for _, profile := range []cbackend.Profile{cbackend.Stage0WindowsMSVCProfile(), cbackend.Stage0Profile()} {
				if _, err := cbackend.EmitStorageTransfers(program, hir.QualifiedModulePath(1, "main")+"::main", profile); err != nil {
					t.Fatal(err)
				}
			}
		})
	}
}
