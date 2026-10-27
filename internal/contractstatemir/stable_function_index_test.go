package contractstatemir

import (
	"fmt"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/packagegraph"
)

func TestStableFunctionRebindKeepsLookupFailuresAndScope(t *testing.T) {
	for _, roots := range []int{1, 2} {
		t.Run(fmt.Sprint(roots), func(t *testing.T) {
			testStableFunctionRebindLookup(t, roots)
		})
	}
}

func testStableFunctionRebindLookup(t *testing.T, roots int) {
	t.Helper()
	input := helperRebindLookupFixture(8, roots)
	coordinates := map[hir.PackageID]packagegraph.CoordinateID{1: {1}}
	artifacts, err := DescribeStableHIRStateBodies(input, []hir.PackageID{1}, coordinates)
	if err != nil {
		t.Fatal(err)
	}
	body := artifacts[1]
	for _, test := range []struct {
		name   string
		mutate func(*hir.Package)
		want   string
	}{
		{"fresh-symbol", func(input *hir.Package) {
			input.Modules[0].Functions[0].Symbol = 200000
		}, ""},
		{"missing", func(input *hir.Package) {
			input.Modules[0].Functions = input.Modules[0].Functions[1:]
		}, "missing stable function"},
		{"duplicate", func(input *hir.Package) {
			input.Modules[0].Functions = append(input.Modules[0].Functions, input.Modules[0].Functions[0])
		}, "multiple functions"},
		{"duplicate-module", func(input *hir.Package) {
			input.Modules = append(input.Modules, hir.Module{Package: 1, Path: "ledger", Functions: []hir.Function{input.Modules[0].Functions[0]}})
		}, "multiple functions"},
		{"other-package", func(input *hir.Package) {
			input.Modules = append(input.Modules, hir.Module{Package: 2, Path: "ledger", Functions: []hir.Function{input.Modules[0].Functions[0]}})
		}, ""},
		{"other-package-only", func(input *hir.Package) {
			input.Modules = append(input.Modules, hir.Module{Package: 2, Path: "ledger", Functions: []hir.Function{input.Modules[0].Functions[0]}})
			input.Modules[0].Functions = input.Modules[0].Functions[1:]
		}, "missing stable function"},
		{"other-module", func(input *hir.Package) {
			input.Modules = append(input.Modules, hir.Module{Package: 1, Path: "other", Functions: []hir.Function{input.Modules[0].Functions[0]}})
		}, ""},
		{"unrelated-duplicate", func(input *hir.Package) {
			input.Modules[0].Functions = append(input.Modules[0].Functions, input.Modules[0].Functions[3])
		}, ""},
		{"distinct-name-components", func(input *hir.Package) {
			input.Modules = append(input.Modules, hir.Module{Package: 1, Functions: []hir.Function{{Name: "ledger::root_0000", Symbol: 300000}}})
		}, ""},
		{"signature-before-later-duplicate", func(input *hir.Package) {
			input.Modules[0].Functions[0].ResultType = hir.TypeI32
			input.Modules[0].Functions = append(input.Modules[0].Functions, input.Modules[0].Functions[1])
		}, "signature does not match"},
	} {
		t.Run(test.name, func(t *testing.T) {
			fresh := helperRebindLookupFixture(8, roots)
			fresh.StateTransactions = nil
			if _, err := RebindStableStateBody(body, body.Hash, fresh, 1, coordinates); err != nil {
				t.Fatal(err)
			}
			test.mutate(&fresh)
			result, err := RebindStableStateBody(body, body.Hash, fresh, 1, coordinates)
			if test.want != "" {
				if err == nil || !strings.Contains(err.Error(), test.want) {
					t.Fatalf("rebind error = %v, want %q", err, test.want)
				}
				return
			}
			if err != nil || len(result.Transactions) != roots {
				t.Fatalf("rebind %d roots: %v", roots, err)
			}
			if result.Transactions[0].Function != fresh.Modules[0].Functions[0].Symbol {
				t.Fatal("rebind used a stale owner symbol")
			}
		})
	}
}

func BenchmarkStableStateOwnerRebindLookup(b *testing.B) {
	for _, roots := range []int{1, 64, 256} {
		b.Run(fmt.Sprint(roots), func(b *testing.B) {
			input := helperRebindLookupFixture(4096, roots)
			coordinates := map[hir.PackageID]packagegraph.CoordinateID{1: {1}}
			artifacts, err := DescribeStableHIRStateBodies(input, []hir.PackageID{1}, coordinates)
			if err != nil {
				b.Fatal(err)
			}
			body := artifacts[1]
			input.StateTransactions = nil
			b.ReportAllocs()
			b.ResetTimer()
			for range b.N {
				result, err := RebindStableStateBody(body, body.Hash, input, 1, coordinates)
				if err != nil || len(result.Transactions) != roots {
					b.Fatalf("rebind %d roots: %v", roots, err)
				}
			}
		})
	}
}
