package semanticsnapshot_test

import (
	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/semanticsnapshot"
	"testing"
)

func TestBuildI64ViewDeclaration(t *testing.T) {
	text := "module api; pub fn ref i64[] tail(ref i64[] values) { return values[1..]; } pub fn i64 first(ref i64[] values) { return values[0]; }"
	result := compileSinglePackage(t, hir.DefaultPackage, "api.fy", text)
	input := packageInput(t, hir.DefaultPackage, "example.i64-view.snapshot", "i64-view", "1.0.0", "api.fy", text, "root")
	snapshot, err := semanticsnapshot.Build(result.Semantic, result.MIR, []semanticsnapshot.PackageInput{input})
	if err != nil {
		t.Fatal(err)
	}
	tail := findDeclaration(t, snapshot.Declarations(), semanticsnapshot.DeclarationFunction, "tail")
	if len(tail.Parameters) != 1 || tail.Parameters[0].Type.Kind != semanticsnapshot.TypeBorrow || tail.Parameters[0].Type.Mutable || len(tail.Parameters[0].Type.Arguments) != 1 || tail.Parameters[0].Type.Arguments[0].Name != hir.TypeI64Slice.String() {
		t.Fatalf("wrong typed-view declaration: %+v", tail)
	}
}
