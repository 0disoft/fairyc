package compiler_test

import (
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/compiler"
	"github.com/0disoft/fairyc/internal/contractstate"
	"github.com/0disoft/fairyc/internal/contractstatemir"
	"github.com/0disoft/fairyc/internal/hir"
)

func TestExperimentalStateGraphSourceHelpers(t *testing.T) {
	data, err := os.ReadFile("../../examples/contract_state_graph/helpers/transaction.fy")
	if err != nil {
		t.Fatal(err)
	}
	for _, reject := range []bool{false, true} {
		t.Run(fmt.Sprint(reject), func(t *testing.T) {
			text := string(data)
			if reject {
				text = strings.Replace(text, `storage::write("value", 42);`, `storage::write("value", 42); storage::write("reject", true);`, 1)
			}
			compiled := compileStateGraph(t, text)
			p := compiled.StateTransactions[0].Program
			if len(p.Helpers) != 2 {
				t.Fatal("missing transitive helpers", p.Helpers)
			}
			store, err := contractstatemir.NewMemoryStore(p.Layout)
			if err != nil {
				t.Fatal(err)
			}
			out, err := contractstatemir.Execute(store, p, contractstate.Limits{MaxWrites: 10, MaxWriteBytes: 4096, MaxValueBytes: 64, MaxMapKeyBytes: 64})
			if err != nil {
				t.Fatal(err)
			}
			if reject {
				if out.Exit != contractstatemir.ExitReject || out.Cost != 6 || len(store.Dump()) != 0 || len(out.Reads) != 1 {
					t.Fatalf("helper rollback %+v", out)
				}
			} else {
				if out.Exit != contractstatemir.ExitCommit || out.Cost != 12 || len(out.Reads) != 3 || len(store.Dump()) != 2 {
					t.Fatalf("helper execution %+v", out)
				}
				value, err := contractstate.U64.Decode(out.Reads[2].Value.Encoded)
				if err != nil || value != 42 || !out.Reads[2].Found {
					t.Fatal("helper did not share root overlay", value, err)
				}
			}
		})
	}
}

func TestExperimentalStateGraphSourceHelperRefusals(t *testing.T) {
	for name, body := range map[string]string{
		"self-recursion":        `fn void helper() { helper(); return; } fn void transaction() { helper(); storage::commit(); }`,
		"mutual-recursion":      `fn void a() { b(); return; } fn void b() { a(); return; } fn void transaction() { a(); storage::commit(); }`,
		"calls-root":            `fn void helper() { transaction(); return; } fn void transaction() { helper(); storage::commit(); }`,
		"unreachable-helper":    `fn void helper() { return; } fn void transaction() { storage::commit(); }`,
		"helper-commits":        `fn void helper() { storage::commit(); } fn void transaction() { helper(); storage::commit(); }`,
		"helper-missing-return": `fn void helper() { } fn void transaction() { helper(); storage::commit(); }`,
		"helper-returns-value":  `fn void helper() { return 1; } fn void transaction() { helper(); storage::commit(); }`,
		"helper-parameters":     `fn void helper(u64 n) { return; } fn void transaction() { helper(1); storage::commit(); }`,
		"call-arguments":        `fn void helper() { return; } fn void transaction() { helper(1); storage::commit(); }`,
		"local-shadow":          `fn void helper() { return; } fn void transaction() { let helper=storage::read("value"); helper(); storage::commit(); }`,
		"missing-effect":        `@effects(storage_write) fn void helper() { storage::write("value",1); return; } fn void transaction() { helper(); storage::commit(); }`,
		"wrong-effect":          `@effects(storage_read) fn void helper() { storage::write("value",1); return; } @effects(storage_read) fn void transaction() { helper(); storage::commit(); }`,
	} {
		t.Run(name, func(t *testing.T) {
			text := `module transaction; @contract_uid("test.state-helpers") state { cell u64 value; } ` + body
			file := parseCompilerFile(t, hir.DefaultPackage, "transaction.fy", text)
			compiled, err := compiler.CheckExperimentalStateGraph(file)
			if compiled.Verified || len(compiled.StateTransactions) != 0 || len(compiled.MIR.Functions) != 0 {
				t.Fatalf("helper refusal published executable: %v %+v", err, compiled)
			}
		})
	}
}

func TestExperimentalStateGraphSourceHelperBounds(t *testing.T) {
	for _, count := range []int{31, 32, 33} {
		var definitions, calls strings.Builder
		for i := 0; i < count; i++ {
			fmt.Fprintf(&definitions, "fn void h%d() { return; }\n", i)
			fmt.Fprintf(&calls, "h%d();", i)
		}
		text := `module transaction; @contract_uid("test.helper-bounds") state { cell u64 value; } ` + definitions.String() + `fn void transaction() { ` + calls.String() + `storage::commit(); }`
		file := parseCompilerFile(t, hir.DefaultPackage, "transaction.fy", text)
		compiled, err := compiler.CheckExperimentalStateGraph(file)
		if (err == nil && compiled.Verified) != (count <= 32) {
			t.Fatal("helper boundary", count, err, compiled.Verified)
		}
	}
	for _, reads := range []int{1022, 1023, 1024} {
		text := `module transaction; @contract_uid("test.helper-bounds") state { cell u64 value; }
@effects(storage_read) fn void helper() { ` + strings.Repeat(`storage::read("value");`, reads) + `return; }
@effects(storage_read) fn void transaction() { helper(); storage::commit(); }`
		file := parseCompilerFile(t, hir.DefaultPackage, "transaction.fy", text)
		compiled, err := compiler.CheckExperimentalStateGraph(file)
		if (err == nil && compiled.Verified) != (reads+1 <= 1024) {
			t.Fatal("aggregate operation boundary", reads+1, err, compiled.Verified)
		}
	}
}
