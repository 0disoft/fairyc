package atlasmap

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/compiler"
	"github.com/0disoft/fairyc/internal/semanticsnapshot"
	"github.com/0disoft/fairyc/internal/workspaceloader"
)

func callBudgetFixture(tb testing.TB, calls int) ([]Partition, map[string]LoadExpectation) {
	tb.Helper()
	root := tb.TempDir()
	if err := os.Mkdir(filepath.Join(root, "src"), 0o700); err != nil {
		tb.Fatal(err)
	}
	manifest := "schema = \"fairyc.manifest/v1\"\n[package]\nuid = \"test.call-budget\"\nname = \"call-budget\"\nversion = \"1.0.0\"\nlanguage = \"fairyc-language/20\"\nsource = \"src\"\n"
	source := "module main;fn i32 leaf(){return 0;}fn i32 main(){" + strings.Repeat("discard leaf();", calls) + "return 0;}"
	for name, contents := range map[string]string{"fairy.toml": manifest, "src/main.fy": source} {
		if err := os.WriteFile(filepath.Join(root, name), []byte(contents), 0o600); err != nil {
			tb.Fatal(err)
		}
	}
	workspace, err := workspaceloader.Load(root)
	if err != nil {
		tb.Fatal(err)
	}
	input, err := workspace.SemanticInput()
	if err != nil {
		tb.Fatal(err)
	}
	result, err := compiler.CheckPackages(input)
	if err != nil || !result.Verified {
		tb.Fatalf("compile: verified=%t, error=%v", result.Verified, err)
	}
	_, partitions, err := PrepareWorkspace(workspace, result, testCompilerBuild())
	if err != nil {
		tb.Fatal(err)
	}
	expectations, err := CurrentWorkspaceExpectations(workspace, result, testCompilerBuild())
	if err != nil {
		tb.Fatal(err)
	}
	return partitions, expectations
}

func BenchmarkWorkspaceCallBudgetRejection(b *testing.B) {
	for _, count := range []int{64, 1024} {
		b.Run(fmt.Sprint(count), func(b *testing.B) {
			partitions, expectations := callBudgetFixture(b, count)
			budget := WorkspaceCallBudget{MaxPackages: 1, MaxCalls: 8}
			b.ReportAllocs()
			b.ResetTimer()
			for range b.N {
				if _, err := NewWorkspaceCallIndex(partitions, expectations, budget); !errors.Is(err, ErrWorkspaceJoinBudget) {
					b.Fatalf("over-budget join: %v", err)
				}
			}
		})
	}
}

func BenchmarkWorkspaceCallBudgetAdmission(b *testing.B) {
	for _, count := range []int{64, 1024} {
		b.Run(fmt.Sprint(count), func(b *testing.B) {
			partitions, expectations := callBudgetFixture(b, count)
			budget := WorkspaceCallBudget{MaxPackages: 1, MaxCalls: count}
			b.ReportAllocs()
			b.ResetTimer()
			for range b.N {
				index, err := NewWorkspaceCallIndex(partitions, expectations, budget)
				if err != nil || index.Usage().Calls != count {
					b.Fatalf("exact-budget join: %v", err)
				}
			}
		})
	}
}

func TestWorkspaceCallBudgetMatchesExpandedLedger(t *testing.T) {
	ordinary, ordinaryExpectations := callBudgetFixture(t, 3)
	generic, genericExpectations := genericCallPartitions(t)
	aggregate, aggregateExpectations := aggregateCallPartitions(t)
	for _, test := range []struct {
		name         string
		partitions   []Partition
		expectations map[string]LoadExpectation
		join         func([]Partition, map[string]LoadExpectation, WorkspaceCallBudget) (*WorkspaceCallIndex, error)
	}{
		{"v1", ordinary, ordinaryExpectations, NewWorkspaceCallIndex},
		{"v2", generic, genericExpectations, NewWorkspaceCallIndexV2},
		{"v3", aggregate, aggregateExpectations, NewWorkspaceCallIndexV3},
	} {
		t.Run(test.name, func(t *testing.T) {
			budget := WorkspaceCallBudget{MaxPackages: len(test.partitions), MaxCalls: 32}
			index, err := test.join(test.partitions, test.expectations, budget)
			if err != nil {
				t.Fatal(err)
			}
			budget.MaxCalls = index.Usage().Calls
			if _, err := test.join(test.partitions, test.expectations, budget); err != nil {
				t.Fatalf("exact call budget rejected: %v", err)
			}
			budget.MaxCalls--
			if _, err := test.join(test.partitions, test.expectations, budget); !errors.Is(err, ErrWorkspaceJoinBudget) {
				t.Fatalf("one-below call budget: %v", err)
			}
		})
	}
}

func TestWorkspaceCallBudgetExcludesForeignCalls(t *testing.T) {
	partitions := []Partition{{data: partitionData{
		functions: []semanticsnapshot.FunctionSummary{{Calls: []semanticsnapshot.Call{
			{TargetKind: semanticsnapshot.CallTargetExternal},
			{TargetKind: semanticsnapshot.CallTargetExternal},
		}}},
	}}}
	if workspaceCallBudgetExceeded(partitions, 0) {
		t.Fatal("foreign calls consumed the resolved-call budget")
	}
	partitions[0].data.functions[0].Calls = append(partitions[0].data.functions[0].Calls,
		semanticsnapshot.Call{TargetKind: semanticsnapshot.CallTargetResolved})
	if !workspaceCallBudgetExceeded(partitions, 0) || workspaceCallBudgetExceeded(partitions, 1) {
		t.Fatal("mixed calls did not consume exactly one resolved-call record")
	}
}

func TestWorkspaceCallBudgetRejectsBeforeIndexAllocation(t *testing.T) {
	partitions, expectations := callBudgetFixture(t, 1024)
	budget := WorkspaceCallBudget{MaxPackages: 1, MaxCalls: 8}
	allocations := testing.AllocsPerRun(5, func() {
		if _, err := NewWorkspaceCallIndex(partitions, expectations, budget); !errors.Is(err, ErrWorkspaceJoinBudget) {
			t.Fatalf("over-budget join: %v", err)
		}
	})
	// Allow small admission/error bookkeeping, never thousands of index allocations.
	if allocations > 16 {
		t.Fatalf("over-budget admission allocated %.0f objects; want at most 16", allocations)
	}
}
