package packageartifact

import (
	"fmt"
	"testing"
)

func TestStableArtifactContextValidationPrecedesBodyValidation(t *testing.T) {
	process := typedProjectionArtifact(t, 4)
	for _, transfers := range []bool{false, true} {
		t.Run(fmt.Sprintf("transfers=%v", transfers), func(t *testing.T) {
			artifact, err := describeStableArtifact(process, stablePackageArtifactLimit, transfers)
			if err != nil {
				t.Fatal(err)
			}
			for _, test := range []struct {
				name   string
				mutate func(*ContextDescriptor)
			}{
				{"unsorted", func(value *ContextDescriptor) {
					value.Structs[0], value.Structs[1] = value.Structs[1], value.Structs[0]
				}},
				{"duplicate", func(value *ContextDescriptor) { value.Structs[1] = value.Structs[0] }},
				{"invalid-field", func(value *ContextDescriptor) { value.Structs[0].Fields[0].Type = "invalid" }},
				{"invalid-drop", func(value *ContextDescriptor) {
					value.Resources = []ContextResource{{Type: value.Structs[0].Type + "Resource", DropKind: "invalid"}}
				}},
				{"invalid-foreign", func(value *ContextDescriptor) {
					value.ForeignFunctions = []ContextForeignFunction{{ABI: "invalid"}}
				}},
			} {
				t.Run(test.name, func(t *testing.T) {
					changed := artifact
					changed.Context = cloneContextDescriptor(artifact.Context)
					test.mutate(&changed.Context)
					// Invalid context must win over missing bodies in both codecs.
					changed.Functions = nil
					want := ValidateContextDescriptor(changed.Context)
					got := validateStableArtifact(changed, transfers)
					if want == nil || got == nil || got.Error() != want.Error() {
						t.Fatalf("context validation changed: got %v, want %v", got, want)
					}
				})
			}
		})
	}
}

func BenchmarkValidateStableArtifactTypedContext(b *testing.B) {
	for _, count := range []int{64, 256} {
		b.Run(fmt.Sprint(count), func(b *testing.B) {
			process := typedProjectionArtifact(b, count)
			// Eight scalar fields per type model a context with nontrivial layouts.
			for i := range process.Context.StructTypes {
				for j := 1; j < 8; j++ {
					field := process.Context.StructTypes[i].Fields[0]
					field.Name = fmt.Sprintf("field_%d", j)
					process.Context.StructTypes[i].Fields = append(process.Context.StructTypes[i].Fields, field)
				}
			}
			var err error
			process.StableContext, err = DescribeContext(process.Context, process.PackageCoordinates, artifactBindings(process))
			if err != nil {
				b.Fatal(err)
			}
			artifact, err := DescribeStableArtifact(process, stablePackageArtifactLimit)
			if err != nil {
				b.Fatal(err)
			}
			b.ReportAllocs()
			b.ResetTimer()
			for range b.N {
				if err := ValidateStableArtifact(artifact); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}
