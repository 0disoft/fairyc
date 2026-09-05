package contractstatenative

import (
	"testing"

	"github.com/0disoft/fairyc/internal/compiler"
	"github.com/0disoft/fairyc/internal/contractstatemir"
	"github.com/0disoft/fairyc/internal/semantic"
	"github.com/0disoft/fairyc/internal/target"
)

func TestPrepareCompiledRejectsContradictoryPublication(t *testing.T) {
	for _, mutation := range []string{"none", "unverified", "missing state", "owner", "value", "ordinary"} {
		t.Run(mutation, func(t *testing.T) {
			input, coordinate := nativeStateInput(t, contractstatemir.ExitCommit)
			result, err := compiler.LowerAnalyzed(compiler.Analysis{Semantic: semantic.Result{HIR: input, CompletedThrough: semantic.PhaseFlow}})
			if err != nil {
				t.Fatal(err)
			}
			switch mutation {
			case "unverified":
				result.Verified = false
			case "missing state":
				result.StateTransactions = nil
			case "owner":
				result.StateTransactions[0].Function++
			case "value":
				result.StateTransactions[0].Program.Operations[0].Value.Encoded[7] = 43
			case "ordinary":
				result.MIR.Functions = nil
			}
			handoff, err := PrepareCompiled(result, 1, 1, coordinate)
			if mutation != "none" {
				if err == nil {
					t.Fatal("contradictory compiler result admitted")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if _, err := Emit(handoff, target.X8664WindowsMSVC); err != nil {
				t.Fatal(err)
			}
		})
	}
}
