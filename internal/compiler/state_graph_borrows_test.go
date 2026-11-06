package compiler_test

import (
	"errors"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/compiler"
	"github.com/0disoft/fairyc/internal/contractstate"
	"github.com/0disoft/fairyc/internal/contractstatemir"
	"github.com/0disoft/fairyc/internal/hir"
)

func TestExperimentalStateGraphSourceBorrows(t *testing.T) {
	data, err := os.ReadFile("../../examples/contract_state_graph/borrows/transaction.fy")
	if err != nil {
		t.Fatal(err)
	}
	for _, enabled := range []bool{false, true} {
		t.Run(fmt.Sprint(enabled), func(t *testing.T) {
			text := string(data)
			if enabled {
				text = strings.Replace(text, `storage::write("source", 42);`, `storage::write("source", 42); storage::write("enabled",true);`, 1)
			}
			compiled := compileStateGraph(t, text)
			p := compiled.StateTransactions[0].Program
			store, err := contractstatemir.NewMemoryStore(p.Layout)
			if err != nil {
				t.Fatal(err)
			}
			out, err := contractstatemir.Execute(store, p, contractstate.Limits{MaxWrites: 10, MaxWriteBytes: 4096, MaxValueBytes: 64, MaxMapKeyBytes: 64})
			if err != nil || out.Exit != contractstatemir.ExitCommit || len(out.Reads) != 3 {
				t.Fatalf("borrow/helper outcome %+v %v", out, err)
			}
			counts := map[uint64]int{}
			for _, entry := range store.Dump() {
				if len(entry.Value) == 8 {
					value, err := contractstate.U64.Decode(entry.Value)
					if err != nil {
						t.Fatal(err)
					}
					counts[value]++
				}
			}
			if counts[42] != 1 || counts[99] != 1 {
				t.Fatal("copied source changed after writing helper", counts)
			}
		})
	}
	for name, body := range map[string]string{
		"map-unrelated-mutation":  `storage::write("values",7,42); let b=storage::borrow("values",7); storage::write("values",8,99); let u64 saved=storage::copy(b); storage::write("value",saved); storage::commit();`,
		"alias-and-presence":      `storage::write("value",42); let b=storage::borrow("value"); let alias=(b); let saved=storage::copy(alias); let bool found=storage::found(b); if (found) { storage::write("values",saved,saved); } storage::commit();`,
		"abandoned-handle":        `let b=storage::borrow("value"); if (storage::found(b)) { storage::read("value"); } storage::commit();`,
		"borrow-inside-found-arm": `let v=storage::read("value"); if (storage::found(v)) { let b=storage::borrow("value"); let saved=storage::copy(b); storage::write("values",saved,saved); } storage::commit();`,
	} {
		t.Run(name, func(t *testing.T) { compileStateGraph(t, stateGraphSource(body)) })
	}
}

func TestExperimentalStateGraphSourceBorrowRefusals(t *testing.T) {
	for name, body := range map[string]string{
		"exact-write-invalidates":   `storage::write("value",42); let b=storage::borrow("value"); storage::write("value",42); let v=storage::copy(b); storage::commit();`,
		"delete-invalidates":        `let b=storage::borrow("value"); storage::delete("value"); let v=storage::copy(b); storage::commit();`,
		"map-exact-write":           `let b=storage::borrow("values",7); storage::write("values",7,99); let v=storage::copy(b); storage::commit();`,
		"map-dynamic-borrow":        `let key=storage::read("value"); let b=storage::borrow("values",key); storage::commit();`,
		"map-dynamic-write-alias":   `let key=storage::read("value"); let b=storage::borrow("values",7); storage::write("values",key,99); storage::commit();`,
		"borrow-as-value":           `let b=storage::borrow("value"); storage::write("value",b); storage::commit();`,
		"borrow-type-annotation":    `let u64 b=storage::borrow("value"); storage::commit();`,
		"view-crosses-edge":         `let b=storage::borrow("value"); let flag=storage::read("flag"); if(flag) { let v=storage::view(b,0,8); } storage::commit();`,
		"copy-crosses-edge":         `let b=storage::borrow("value"); if(storage::found(b)) { let v=storage::copy(b); } storage::commit();`,
		"borrow-alias-crosses-edge": `let b=storage::borrow("value"); let alias=b; if(storage::found(b)) {} let v=storage::copy(alias); storage::commit();`,
		"view-zero-length":          `let b=storage::borrow("value"); let v=storage::view(b,0,0); storage::commit();`,
		"view-negative-offset":      `let b=storage::borrow("value"); let v=storage::view(b,-1,8); storage::commit();`,
		"view-overflow-offset":      `let b=storage::borrow("value"); let v=storage::view(b,4294967296,8); storage::commit();`,
		"view-overflow-length":      `let b=storage::borrow("value"); let v=storage::view(b,0,4294967296); storage::commit();`,
		"view-dynamic-bounds":       `let n=storage::read("value"); let b=storage::borrow("value"); let v=storage::view(b,n,8); storage::commit();`,
		"copy-not-borrowed":         `let b=storage::read("value"); let v=storage::copy(b); storage::commit();`,
		"view-wrong-count":          `let b=storage::borrow("value"); let v=storage::view(b,0); storage::commit();`,
		"borrow-missing-field":      `let b=storage::borrow("missing"); storage::commit();`,
	} {
		t.Run(name, func(t *testing.T) {
			file := parseCompilerFile(t, hir.DefaultPackage, "transaction.fy", stateGraphSource(body))
			compiled, err := compiler.CheckExperimentalStateGraph(file)
			if compiled.Verified || len(compiled.StateTransactions) != 0 || len(compiled.MIR.Functions) != 0 {
				t.Fatalf("borrow refusal published executable %v %+v", err, compiled)
			}
		})
	}
	data, err := os.ReadFile("../../examples/contract_state_graph/borrows/transaction.fy")
	if err != nil {
		t.Fatal(err)
	}
	for name, text := range map[string]string{
		"helper-owns-borrow":       strings.Replace(string(data), `storage::read("source");`, `let b=storage::borrow("source");`, 1),
		"writing-call-invalidates": strings.Replace(string(data), "inspect_source();", "replace_source();", 1),
	} {
		t.Run(name, func(t *testing.T) {
			file := parseCompilerFile(t, hir.DefaultPackage, "transaction.fy", text)
			compiled, err := compiler.CheckExperimentalStateGraph(file)
			if compiled.Verified || len(compiled.StateTransactions) != 0 || len(compiled.MIR.Functions) != 0 {
				t.Fatalf("helper borrow invalidation %v %+v", err, compiled)
			}
		})
	}
}

func TestExperimentalStateGraphSourceBorrowRuntimeBounds(t *testing.T) {
	for _, trial := range []struct {
		name, body         string
		valid, invalidated bool
	}{
		{"missing-borrow", `let b=storage::borrow("value"); let v=storage::copy(b); storage::commit();`, false, true},
		{"exact-view", `storage::write("value",42); let b=storage::borrow("value"); let v=storage::view(b,0,8); let saved=storage::copy(v); storage::write("values",7,saved); storage::commit();`, true, false},
		{"past-end-view", `storage::write("value",42); let b=storage::borrow("value"); let v=storage::view(b,1,8); storage::commit();`, false, false},
		{"maximum-offset", `storage::write("value",42); let b=storage::borrow("value"); let v=storage::view(b,4294967295,1); storage::commit();`, false, false},
		{"maximum-length", `storage::write("value",42); let b=storage::borrow("value"); let v=storage::view(b,0,4294967295); storage::commit();`, false, false},
		{"noncanonical-fixed-copy", `storage::write("value",42); let b=storage::borrow("value"); let v=storage::view(b,1,7); let saved=storage::copy(v); storage::write("value",saved); storage::commit();`, false, false},
	} {
		t.Run(trial.name, func(t *testing.T) {
			compiled := compileStateGraph(t, stateGraphSource(trial.body))
			p := compiled.StateTransactions[0].Program
			store, err := contractstatemir.NewMemoryStore(p.Layout)
			if err != nil {
				t.Fatal(err)
			}
			_, err = contractstatemir.Execute(store, p, contractstate.Limits{MaxWrites: 10, MaxWriteBytes: 4096, MaxValueBytes: 64, MaxMapKeyBytes: 64})
			if (err == nil) != trial.valid || trial.invalidated && !errors.Is(err, contractstate.ErrBorrowInvalidated) {
				t.Fatal("borrow runtime result", err)
			}
			if !trial.valid && len(store.Dump()) != 0 {
				t.Fatal("failed borrow published writes")
			}
		})
	}
}
