package compiler_test

import (
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/compiler"
	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/semantic"
)

const stateSource = `module transaction;
@contract_uid("example.native-state")
state { cell u64 value; }
@effects(storage_read, storage_write)
fn void transaction() {
    storage::write("value", 42);
    storage::read("value");
    storage::commit();
}`

func TestExperimentalStateSource(t *testing.T) {
	file := parseCompilerFile(t, hir.DefaultPackage, "transaction.fy", stateSource)
	result, err := compiler.CheckExperimentalState(file)
	if err != nil || !result.Verified || len(result.StateTransactions) != 1 {
		t.Fatalf("source publication: %v %+v", err, result)
	}
	if len(file.Module.Functions[0].Body.Statements) != 3 {
		t.Fatal("source AST mutated")
	}
	ordinary, err := compiler.Check([]semantic.File{file})
	if err == nil && ordinary.Verified {
		t.Fatal("ordinary compiler silently admitted experimental operations")
	}
	for _, replacement := range []struct{ from, to string }{
		{"storage::commit();", ""},
		{"storage::commit();", "storage::commit(); storage::read(\"value\");"},
		{"42", "1 + 41"},
		{"storage::write(\"value\", 42)", "storage::write(\"missing\", 42)"},
		{"storage_read, storage_write", "storage_read"},
	} {
		file := parseCompilerFile(t, hir.DefaultPackage, "transaction.fy", strings.Replace(stateSource, replacement.from, replacement.to, 1))
		result, err := compiler.CheckExperimentalState(file)
		if err == nil && result.Verified {
			t.Fatalf("admitted mutation %+v", replacement)
		}
	}
}
