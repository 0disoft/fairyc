package packageartifact

import (
	"bytes"
	"fmt"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/mir"
)

func TestUnusedForeignProjectionPreservesBytesAndValidation(t *testing.T) {
	for _, tc := range []struct {
		name     string
		describe func(Artifact, uint64) (StableArtifact, error)
		encode   func(StableArtifact, uint64) ([]byte, error)
	}{
		{"ordinary", DescribeStableArtifact, EncodeStableArtifact},
		{"storage transfers", DescribeStorageTransferStableArtifact, EncodeStorageTransferStableArtifact},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var expected []byte
			for _, count := range []int{0, 64} {
				artifact := unusedForeignProjectionFixture(t, count)
				stable, err := tc.describe(artifact, stablePackageArtifactLimit)
				if err != nil {
					t.Fatal(err)
				}
				encoded, err := tc.encode(stable, stablePackageArtifactLimit)
				if err != nil {
					t.Fatal(err)
				}
				if count == 0 {
					expected = encoded
				} else if !bytes.Equal(expected, encoded) {
					t.Fatal("unused declarations changed stable artifact bytes")
				}
			}
			artifact := unusedForeignProjectionFixture(t, 1)
			artifact.Context.ForeignFunctions[0].ABI = "invalid"
			if _, err := tc.describe(artifact, stablePackageArtifactLimit); err == nil || !strings.Contains(err.Error(), "verify process-local artifact before stable projection") {
				t.Fatalf("invalid unused declaration bypassed full artifact validation: %v", err)
			}
		})
	}
}

func BenchmarkDescribeArtifactUnusedForeign(b *testing.B) {
	for _, count := range []int{64, 1024, 4096} {
		b.Run(fmt.Sprint(count), func(b *testing.B) {
			artifact := unusedForeignProjectionFixture(b, count)
			b.ReportAllocs()
			b.ResetTimer()
			for range b.N {
				stable, err := DescribeStableArtifact(artifact, stablePackageArtifactLimit)
				if err != nil || len(stable.Context.ForeignFunctions) != 0 {
					b.Fatalf("project artifact with %d unused declarations: %v", count, err)
				}
			}
		})
	}
}

func unusedForeignProjectionFixture(t testing.TB, count int) Artifact {
	t.Helper()
	program, identities, coordinates := extractionContextFixture(1, 1)
	span := program.Functions[0].Span
	span.Source = "main.fy"
	program.Functions[0].Span = span
	for i := range count {
		name := fmt.Sprintf("external_%06d", i)
		program.ForeignFunctions = append(program.ForeignFunctions, mir.ForeignFunction{
			Symbol: hir.SymbolID(20000 + i), Name: hir.QualifiedModulePath(span.Package, "ffi::"+name), Span: span,
			LinkName: name, ABI: "C", ParameterTypes: []hir.Type{hir.TypeI32, hir.TypeUSize}, ResultType: hir.TypeVoid,
		})
	}
	artifact, err := Extract(program, span.Package, identities, coordinates)
	if err != nil {
		t.Fatal(err)
	}
	return artifact
}
