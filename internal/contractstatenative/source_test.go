package contractstatenative

import (
	"os"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/compiler"
	"github.com/0disoft/fairyc/internal/contractstatemir"
	"github.com/0disoft/fairyc/internal/frontend"
	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/packagegraph"
	"github.com/0disoft/fairyc/internal/semantic"
	"github.com/0disoft/fairyc/internal/source"
	"github.com/0disoft/fairyc/internal/target"
)

func nativeSourceResult(t *testing.T, exit contractstatemir.ExitKind, deleting bool) (compiler.Result, packagegraph.CoordinateID) {
	t.Helper()
	data, err := os.ReadFile("../../examples/contract_state_native/transaction.fy")
	if err != nil {
		t.Fatal(err)
	}
	text := string(data)
	if deleting {
		text = strings.Replace(text, `storage::write("value", 42)`, `storage::delete("value")`, 1)
	}
	switch exit {
	case contractstatemir.ExitReject:
		text = strings.Replace(text, "storage::commit()", "storage::reject()", 1)
	case contractstatemir.ExitTrap:
		text = strings.Replace(text, "storage::commit()", `storage::trap("overflow")`, 1)
	}
	parsed, err := frontend.AnalyzeSyntax(source.NewFile("transaction.fy", []byte(text)), source.DefaultLimits())
	if err != nil || parsed.AST == nil {
		t.Fatalf("parse state example: %v", err)
	}
	result, err := compiler.CheckExperimentalState(semantic.File{Package: hir.DefaultPackage, Path: "transaction.fy", Module: parsed.AST})
	if err != nil || !result.Verified {
		t.Fatalf("compile state example: %v %+v", err, result.Semantic.Diagnostics)
	}
	coordinate, err := packagegraph.ParseCoordinateID("fypc1-sha256-c14ce14f76ca2cc42c89dfb70b90c251684bbb445d6c659c3956eeed097efd67")
	if err != nil {
		t.Fatal(err)
	}
	return result, coordinate
}

func TestSourceStateExampleEmitsDeterministically(t *testing.T) {
	result, coordinate := nativeSourceResult(t, contractstatemir.ExitCommit, false)
	owner := result.StateTransactions[0]
	handoff, err := PrepareCompiled(result, owner.Package, owner.Function, coordinate)
	if err != nil {
		t.Fatal(err)
	}
	first, err := Emit(handoff, target.X8664WindowsMSVC)
	if err != nil {
		t.Fatal(err)
	}
	second, err := Emit(handoff, target.X8664WindowsMSVC)
	if err != nil || first != second {
		t.Fatalf("unstable source emission: %v", err)
	}
}
