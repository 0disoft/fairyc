package packageartifact

import (
	"bytes"
	"errors"
	"fmt"
	"testing"

	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/mir"
)

func TestStableProjectionContextMatchesIndependentBodies(t *testing.T) {
	artifact := typedProjectionArtifact(t, 4)
	stable, err := DescribeStableArtifact(artifact, stablePackageArtifactLimit)
	if err != nil {
		t.Fatal(err)
	}
	for i, record := range artifact.Functions {
		independent, err := DescribeStableLeafBody(record, stable.Context, artifact.PackageCoordinates, stablePackageArtifactLimit)
		if err != nil {
			t.Fatal(err)
		}
		want, err := EncodeStableBody(independent, stablePackageArtifactLimit)
		if err != nil {
			t.Fatal(err)
		}
		got, err := EncodeStableBody(stable.Functions[i], stablePackageArtifactLimit)
		if err != nil || !bytes.Equal(got, want) {
			t.Fatalf("batch body %d differs from independently projected body: %v", i, err)
		}
	}
	contextBytes, err := EncodeContextDescriptor(stable.Context, stablePackageArtifactLimit)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := DescribeStableArtifact(artifact, uint64(len(contextBytes)-1)); !errors.Is(err, ErrContextArtifactTooLarge) {
		t.Fatalf("context byte limit was not enforced: %v", err)
	}
	// A later projection must observe changed type facts, even with the same
	// function identities and byte limit.
	changed := cloneArtifactForTest(artifact)
	changed.Context.StructTypes[0].Fields[0].Type = hir.TypeUSize
	changed.StableContext, err = DescribeContext(changed.Context, changed.PackageCoordinates, artifactBindings(changed))
	if err != nil {
		t.Fatal(err)
	}
	updated, err := DescribeStableArtifact(changed, stablePackageArtifactLimit)
	if err != nil {
		t.Fatal(err)
	}
	for i, body := range updated.Functions {
		if body.ContextID == stable.Functions[i].ContextID {
			t.Fatalf("body %d reused the previous projection's context ID", i)
		}
	}
}

func BenchmarkDescribeStableArtifactTypedContext(b *testing.B) {
	for _, count := range []int{64, 256} {
		b.Run(fmt.Sprint(count), func(b *testing.B) {
			artifact := typedProjectionArtifact(b, count)
			b.ReportAllocs()
			b.ResetTimer()
			for range b.N {
				stable, err := DescribeStableArtifact(artifact, stablePackageArtifactLimit)
				if err != nil || len(stable.Functions) != count || len(stable.Context.Structs) != count {
					b.Fatalf("project %d functions and types: %v", count, err)
				}
			}
		})
	}
}

func typedProjectionArtifact(t testing.TB, count int) Artifact {
	t.Helper()
	_, program, identities, coordinates := checkedWorkspaceProgram(t)
	span := program.Functions[0].Span
	prefix := hir.QualifiedModulePath(span.Package, "main")
	program.Functions = make([]mir.Function, count)
	program.StructTypes = make([]mir.StructType, count)
	for i := range count {
		name := fmt.Sprintf("Type_%d", i)
		typeValue := hir.ParseType(prefix + "::" + name)
		program.StructTypes[i] = mir.StructType{
			Type: typeValue, Name: name, Span: span,
			Fields: []mir.StructField{{Name: "value", Type: hir.TypeI32, Span: span}},
		}
		program.Functions[i] = mir.Function{
			Symbol: hir.SymbolID(10000 + i), Name: fmt.Sprintf("%s::bench_%d", prefix, i), Span: span,
			Parameters: []mir.LocalID{mir.LocalID(10000 + i)}, ParameterTypes: []hir.Type{typeValue},
			ResultType: hir.TypeI32, Entry: 0,
			Blocks: []mir.Block{{ID: 0, Terminator: mir.Terminator{
				Kind: mir.TerminatorReturn, UnwindSuccessor: mir.NoBlock,
				ReturnValue: mir.Operand{Kind: mir.OperandI32, Type: hir.TypeI32, I32: int32(i)},
			}}},
		}
	}
	artifact, err := Extract(program, span.Package, identities, coordinates)
	if err != nil {
		t.Fatal(err)
	}
	return artifact
}
