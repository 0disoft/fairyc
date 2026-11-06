package compiler_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"os"
	"reflect"
	"testing"

	"github.com/0disoft/fairyc/internal/compiler"
	"github.com/0disoft/fairyc/internal/contractstatemir"
	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/packagegraph"
	"github.com/0disoft/fairyc/internal/semantic"
)

func TestExperimentalStateGraphSourceAnalysis(t *testing.T) {
	coordinate, err := packagegraph.ComputeCoordinateID(packagegraph.Coordinate{PackageUID: "test.state-analysis", VersionIdentity: "1"})
	if err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{"transaction.fy", "helpers/transaction.fy", "borrows/transaction.fy", "dynamic/transaction.fy"} {
		t.Run(path, func(t *testing.T) {
			data, err := os.ReadFile("../../examples/contract_state_graph/" + path)
			if err != nil {
				t.Fatal(err)
			}
			file := parseCompilerFile(t, 57, "transaction.fy", string(data))
			before, err := json.Marshal(file.Module)
			if err != nil {
				t.Fatal(err)
			}
			analysis, err := compiler.AnalyzeExperimentalStateGraph(file)
			if err != nil || analysis.Semantic.CompletedThrough != semantic.PhaseFlow || len(analysis.Semantic.HIR.StateTransactions) == 0 {
				t.Fatal("current source analysis", err, analysis.Semantic.Diagnostics)
			}
			lowered, err := compiler.LowerAnalyzed(analysis)
			if err != nil || !lowered.Verified || len(lowered.StateTransactions) != 1 {
				t.Fatal("analyzed state did not publish atomically", err)
			}
			checked, err := compiler.CheckExperimentalStateGraph(file)
			if err != nil || !checked.Verified || !reflect.DeepEqual(checked.MIR, lowered.MIR) {
				t.Fatal("check and analyze/lower differ", err)
			}
			coordinates := map[hir.PackageID]packagegraph.CoordinateID{57: coordinate}
			got, err := contractstatemir.ComputeHIRStateBodyHashes(lowered.Semantic.HIR, []hir.PackageID{57}, coordinates)
			want, wantErr := contractstatemir.ComputeHIRStateBodyHashes(checked.Semantic.HIR, []hir.PackageID{57}, coordinates)
			if err != nil || wantErr != nil || got[57] == (contractstatemir.StateBodyHash{}) || !reflect.DeepEqual(got, want) {
				t.Fatal("source body identity changed across analysis/lowering", err, wantErr)
			}
			after, err := json.Marshal(file.Module)
			if err != nil || !bytes.Equal(before, after) {
				t.Fatal("analysis/lowering mutated current source", err)
			}
			ctx, cancel := context.WithCancel(context.Background())
			cancel()
			if result, err := compiler.LowerAnalyzedContext(ctx, analysis); !errors.Is(err, context.Canceled) || result.Verified || len(result.MIR.Functions)+len(result.StateTransactions) != 0 {
				t.Fatal("canceled analysis published MIR", err)
			}
		})
	}
}

func TestExperimentalStateGraphSourceAnalysisFailures(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if result, err := compiler.AnalyzeExperimentalStateGraphContext(ctx, semantic.File{}); !errors.Is(err, context.Canceled) || len(result.Semantic.HIR.StateTransactions) != 0 {
		t.Fatal("cancel must precede source admission", err)
	}
	if result, err := compiler.CheckExperimentalStateGraphContext(ctx, semantic.File{}); !errors.Is(err, context.Canceled) || result.Verified {
		t.Fatal("canceled check published state", err)
	}
	if _, err := compiler.AnalyzeExperimentalStateGraphContext(nil, semantic.File{}); err == nil {
		t.Fatal("nil context accepted")
	}
	if _, err := compiler.AnalyzeExperimentalStateGraph(semantic.File{}); err == nil {
		t.Fatal("missing source accepted")
	}
	file := parseCompilerFile(t, hir.DefaultPackage, "transaction.fy", dynamicStateGraphSource("text", `storage::write("value",b"wrong");storage::commit();`))
	if result, err := compiler.AnalyzeExperimentalStateGraph(file); err == nil || len(result.Semantic.HIR.StateTransactions) != 0 {
		t.Fatal("invalid operand escaped analysis", err)
	}
}
