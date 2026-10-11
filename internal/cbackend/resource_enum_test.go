package cbackend_test

import (
	"slices"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/cbackend"
)

func TestEmitResourceEnumFieldLayouts(t *testing.T) {
	t.Parallel()
	for _, test := range []struct{ name, declaration, value, arms string }{
		{"tag", "enum State { Off; On; }", ".On", "case .Off : { 0 } case .On : { 42 }"},
		{"payload", "enum State { Empty; Value { i32 amount; }; }", ".Value { .amount = 42 }", "case .Empty : { 0 } case .Value { .amount = n } : { n }"},
	} {
		t.Run(test.name, func(t *testing.T) {
			program := lowerSource(t, "module main;\n"+test.declaration+"\nresource Holder { State state; cleanup {} }\nfn i32 main(){ let Holder holder = Holder { .state = "+test.value+" }; return switch(holder.state){"+test.arms+"}; }\n")
			for _, profile := range []cbackend.Profile{cbackend.Stage0Profile(), cbackend.Stage0WindowsMSVCProfile()} {
				resourceLayout, e := cbackend.StorageLayout(program, program.ResourceTypes[0].Type, profile)
				if e != nil {
					t.Fatal(e)
				}
				enumLayout, e := cbackend.StorageLayout(program, program.EnumTypes[0].Type, profile)
				if e != nil || resourceLayout.SizeBytes != enumLayout.SizeBytes || resourceLayout.AlignmentBytes != enumLayout.AlignmentBytes {
					t.Fatalf("resource/enum storage mismatch: %+v %+v %v", resourceLayout, enumLayout, e)
				}
				unit, e := cbackend.Emit(program, "main::main", profile)
				if e != nil {
					t.Fatal(e)
				}
				if !slices.Contains(unit.Features, "nested-copy-resource") {
					t.Fatalf("missing enum resource layout feature: %v", unit.Features)
				}
				for _, part := range []string{"fy_enum_0 f_0;", "_Static_assert(sizeof(fy_resource_0)", "offsetof(fy_resource_0, f_0) == UINT64_C(0)"} {
					if !strings.Contains(unit.Source, part) {
						t.Fatalf("missing enum field/layout %q", part)
					}
				}
				if strings.Index(unit.Source, "} fy_enum_0;") > strings.Index(unit.Source, "} fy_resource_0;") {
					t.Fatal("resource emitted before its enum field type")
				}
			}
		})
	}
}
