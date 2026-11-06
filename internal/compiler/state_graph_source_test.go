package compiler_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/compiler"
	"github.com/0disoft/fairyc/internal/contractstate"
	"github.com/0disoft/fairyc/internal/contractstatemir"
	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/semantic"
)

func stateGraphSource(body string) string {
	return `module transaction;
@contract_uid("test.source-state-graph")
state { cell u64 value; cell bool flag; map u64 u64 values; }
@effects(storage_read, storage_write)
fn void transaction() { ` + body + ` }`
}

func compileStateGraph(t *testing.T, text string) compiler.Result {
	t.Helper()
	file := parseCompilerFile(t, hir.DefaultPackage, "transaction.fy", text)
	before, err := json.Marshal(file.Module)
	if err != nil {
		t.Fatal(err)
	}
	result, err := compiler.CheckExperimentalStateGraph(file)
	if err != nil || !result.Verified || len(result.StateTransactions) != 1 {
		t.Fatalf("graph source rejected: %v; semantic %+v", err, result.Semantic)
	}
	after, err := json.Marshal(file.Module)
	if err != nil || !bytes.Equal(before, after) {
		t.Fatal("source AST changed", err)
	}
	if failures := contractstatemir.Verify(result.StateTransactions[0].Program); len(failures) != 0 {
		t.Fatal(failures)
	}
	if old, err := compiler.CheckExperimentalState(file); err == nil && old.Verified {
		t.Fatal("original single-cell source admission widened")
	}
	if ordinary, err := compiler.Check([]semantic.File{file}); err == nil && ordinary.Verified {
		t.Fatal("ordinary source admission widened")
	}
	return result
}

func TestExperimentalStateGraphSource(t *testing.T) {
	data, err := os.ReadFile("../../examples/contract_state_graph/transaction.fy")
	if err != nil {
		t.Fatal(err)
	}
	result := compileStateGraph(t, string(data))
	program := result.StateTransactions[0].Program
	if len(program.Blocks) != 7 || len(program.Layout.Fields) != 4 || len(program.Operations) != 0 {
		t.Fatalf("wrong graph shape: %+v", program)
	}
	limits := contractstate.Limits{MaxWrites: 10, MaxWriteBytes: 4096, MaxValueBytes: 64, MaxMapKeyBytes: 32}
	for _, trial := range []struct {
		name   string
		prefix string
		want   uint64
		cost   uint64
	}{
		{"disabled", "", 0, 10},
		{"enabled-missing", `storage::write("enabled", true);`, 42, 13},
		{"enabled-found", `storage::write("enabled", true); storage::write("values", 0, 73);`, 73, 14},
	} {
		t.Run(trial.name, func(t *testing.T) {
			text := strings.Replace(string(data), `let key =`, trial.prefix+` let key =`, 1)
			compiled := compileStateGraph(t, text)
			p := compiled.StateTransactions[0].Program
			layout := sourceGraphLayout(t, p.Layout)
			store, err := contractstate.NewMemoryStore(layout)
			if err != nil {
				t.Fatal(err)
			}
			out, err := contractstatemir.Execute(store, p, limits)
			if err != nil || out.Exit != contractstatemir.ExitCommit || len(out.Reads) != 5 || out.Cost != trial.cost {
				t.Fatalf("outcome: %+v %v", out, err)
			}
			got, err := contractstate.U64.Decode(out.Reads[len(out.Reads)-1].Value.Encoded)
			if err != nil || got != trial.want || !out.Reads[len(out.Reads)-1].Found {
				t.Fatalf("selected map value %d/%v; want %d", got, err, trial.want)
			}
		})
	}
}

func sourceGraphLayout(t *testing.T, input contractstatemir.Layout) contractstate.Layout {
	t.Helper()
	builder, err := contractstate.NewLayoutBuilder(input.ContractUID)
	if err != nil {
		t.Fatal(err)
	}
	for _, field := range input.Fields {
		if field.Kind == contractstate.FieldCell {
			_, err = contractstate.DeclareEncodedCell(builder, field.Name, field.ValueTypeID)
		} else {
			_, err = contractstate.DeclareEncodedMap(builder, field.Name, field.KeyTypeID, field.ValueTypeID)
		}
		if err != nil {
			t.Fatal(err)
		}
	}
	layout, err := builder.Build()
	if err != nil || layout.Hash() != input.Hash {
		t.Fatal("layout mismatch", err)
	}
	return layout
}

func TestExperimentalStateGraphSourceEdges(t *testing.T) {
	for name, body := range map[string]string{
		"one-returning-arm": `let v = storage::read("value"); if (storage::found(v)) { storage::reject(); } storage::write("value", v); storage::commit();`,
		"two-closed-arms":   `let v = storage::read("flag"); if (v) { storage::commit(); } else { storage::reject(); }`,
		"found-alias":       `let v = storage::read("value"); let bool present = storage::found(v); if (present) { storage::write("value", v); } storage::commit();`,
		"typed-alias":       `let u64 v = storage::read("value"); let u64 saved = v; if (storage::found(saved)) { storage::write("values", saved, v); } storage::commit();`,
	} {
		t.Run(name, func(t *testing.T) { compileStateGraph(t, stateGraphSource(body)) })
	}
}

func TestExperimentalStateGraphSourceRefusals(t *testing.T) {
	for _, body := range []string{
		`storage::read("value");`,
		`storage::commit(); storage::read("value");`,
		`storage::write("value", 18446744073709551616); storage::commit();`,
		`storage::write("value", -1); storage::commit();`,
		`storage::write("flag", 1); storage::commit();`,
		`storage::write("missing", 1); storage::commit();`,
		`storage::write("values", 1); storage::commit();`,
		`storage::read("value", 1); storage::commit();`,
		`storage::read("values"); storage::commit();`,
		`let value = storage::read("value"); storage::write("flag", value); storage::commit();`,
		`let value = storage::read("flag"); storage::write("values", value, 1); storage::commit();`,
		`var value = storage::read("value"); storage::commit();`,
		`let u32 value = storage::read("value"); storage::commit();`,
		`let value = storage::read("value"); let value = storage::read("value"); storage::commit();`,
		`let value = storage::read("flag"); if (value) { let nested = storage::read("value"); } storage::write("value", nested); storage::commit();`,
		`if (true) { storage::commit(); } else { storage::reject(); }`,
		`let value = storage::read("value"); if (value) { storage::commit(); } else { storage::reject(); }`,
		`let value = storage::read("flag"); if (value) { storage::commit(); } storage::read("value");`,
		`storage::commit(1);`, `storage::trap(1);`, `storage::trap("Bad trap");`,
		`let value = storage::found(unknown); storage::commit();`,
		`unknown(); storage::commit();`, `return;`,
		`while (true) { storage::read("value"); } storage::commit();`,
		`let value = storage::read("value"); value = 1; storage::commit();`,
		`let value = storage::read("value"); storage::write("value", value + 1); storage::commit();`,
	} {
		t.Run(fmt.Sprintf("%02d", len(body))+body[:min(len(body), 45)], func(t *testing.T) {
			file := parseCompilerFile(t, hir.DefaultPackage, "transaction.fy", stateGraphSource(body))
			result, err := compiler.CheckExperimentalStateGraph(file)
			if result.Verified || len(result.MIR.Functions) != 0 || len(result.StateTransactions) != 0 {
				t.Fatalf("invalid source published executable state: %v %+v", err, result)
			}
		})
	}
	for _, change := range []struct{ from, to string }{
		{"storage_read, storage_write", "storage_read"},
		{`cell u64 value`, `cell u64 value; cell u64 value`},
		{`fn void transaction()`, `pub fn void transaction()`},
		{`fn void transaction()`, `unsafe fn void transaction()`},
		{`fn void transaction()`, `fn u64 transaction()`},
		{`fn void transaction()`, `fn void other()`},
		{`fn void transaction()`, `@inline fn void transaction()`},
	} {
		text := strings.Replace(stateGraphSource(`storage::write("value", 1); storage::commit();`), change.from, change.to, 1)
		file := parseCompilerFile(t, hir.DefaultPackage, "transaction.fy", text)
		result, err := compiler.CheckExperimentalStateGraph(file)
		if err == nil && result.Verified {
			t.Fatalf("invalid owner accepted: %+v", change)
		}
	}
}

func TestExperimentalStateGraphSourceCodecs(t *testing.T) {
	for _, trial := range []struct {
		codec, literal string
		valid          bool
	}{
		{"bool", "true", true}, {"bool", "false", true}, {"bool", "2", false},
		{"u32", "0", true}, {"u32", "4294967294", true}, {"u32", "4294967295", true}, {"u32", "4294967296", false},
		{"u64", "0", true}, {"u64", "18446744073709551614", true}, {"u64", "18446744073709551615", true}, {"u64", "18446744073709551616", false},
		{"bytes20", `b"` + strings.Repeat(`\x00`, 19) + `"`, false},
		{"bytes20", `b"` + strings.Repeat(`\xff`, 20) + `"`, true},
		{"bytes20", `b"` + strings.Repeat(`\x00`, 21) + `"`, false},
		{"bytes32", `b"` + strings.Repeat(`\x00`, 31) + `"`, false},
		{"bytes32", `b"` + strings.Repeat(`\xff`, 32) + `"`, true},
		{"bytes32", `b"` + strings.Repeat(`\x00`, 33) + `"`, false},
		{"bytes20", `"` + strings.Repeat("a", 20) + `"`, false},
	} {
		t.Run(trial.codec+"/"+trial.literal, func(t *testing.T) {
			text := fmt.Sprintf(`module transaction;
@contract_uid("test.source-state-codec")
state { cell %s value; map %s %s values; }
@effects(storage_read, storage_write)
fn void transaction() {
 storage::write("value", %s);
 let %s value = storage::read("value");
 storage::write("values", value, value);
 storage::read("values", value);
 storage::delete("values", value);
 storage::commit();
}`, trial.codec, trial.codec, trial.codec, trial.literal, trial.codec)
			file := parseCompilerFile(t, hir.DefaultPackage, "transaction.fy", text)
			result, err := compiler.CheckExperimentalStateGraph(file)
			if (err == nil && result.Verified) != trial.valid {
				t.Fatalf("literal admission: %v verified=%v", err, result.Verified)
			}
			if !trial.valid {
				return
			}
			p := result.StateTransactions[0].Program
			store, err := contractstatemir.NewMemoryStore(p.Layout)
			if err != nil {
				t.Fatal(err)
			}
			out, err := contractstatemir.Execute(store, p, contractstate.Limits{MaxWrites: 10, MaxWriteBytes: 4096, MaxValueBytes: 64, MaxMapKeyBytes: 64})
			if err != nil || out.Exit != contractstatemir.ExitCommit || len(out.Reads) != 2 || !out.Reads[0].Found || !out.Reads[1].Found || !bytes.Equal(out.Reads[0].Value.Encoded, out.Reads[1].Value.Encoded) {
				t.Fatalf("typed dynamic key/value: %+v %v", out, err)
			}
		})
	}
}

func TestExperimentalStateGraphSourceBounds(t *testing.T) {
	for _, operations := range []int{1023, 1024, 1025} {
		text := stateGraphSource(strings.Repeat(`storage::read("value");`, operations) + `storage::commit();`)
		file := parseCompilerFile(t, hir.DefaultPackage, "transaction.fy", text)
		result, err := compiler.CheckExperimentalStateGraph(file)
		if (err == nil && result.Verified) != (operations <= 1024) {
			t.Fatalf("operation boundary %d: %v verified=%v", operations, err, result.Verified)
		}
	}
	// Each continuing if adds two arms and one join, on top of entry.
	for _, branches := range []int{20, 21, 22} {
		text := stateGraphSource(`let flag = storage::read("flag");` + strings.Repeat(`if (flag) { storage::read("value"); }`, branches) + `storage::commit();`)
		file := parseCompilerFile(t, hir.DefaultPackage, "transaction.fy", text)
		result, err := compiler.CheckExperimentalStateGraph(file)
		if (err == nil && result.Verified) != (branches <= 21) {
			t.Fatalf("block boundary %d: %v verified=%v", 1+3*branches, err, result.Verified)
		}
	}
}
